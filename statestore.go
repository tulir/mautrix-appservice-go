// Copyright (c) 2019 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package appservice

import (
	"sync"
	"time"

	"maunium.net/go/mautrix"
)

type StateStore interface {
	IsRegistered(userID string) bool
	MarkRegistered(userID string)

	IsTyping(roomID, userID string) bool
	SetTyping(roomID, userID string, timeout int64)

	IsInRoom(roomID, userID string) bool
	IsInvited(roomID, userID string) bool
	IsMembership(roomID, userID string, allowedMemberships ...mautrix.Membership) bool
	SetMembership(roomID, userID string, membership mautrix.Membership)

	SetPowerLevels(roomID string, levels *mautrix.PowerLevels)
	GetPowerLevels(roomID string) *mautrix.PowerLevels
	GetPowerLevel(roomID, userID string) int
	GetPowerLevelRequirement(roomID string, eventType mautrix.EventType) int
	HasPowerLevel(roomID, userID string, eventType mautrix.EventType) bool

	Lock()
	Unlock()
	RLock()
	RUnlock()
}

func (as *AppService) UpdateState(evt *mautrix.Event) {
	switch evt.Type {
	case mautrix.StateMember:
		as.StateStore.SetMembership(evt.RoomID, evt.GetStateKey(), evt.Content.Membership)
	case mautrix.StatePowerLevels:
		as.StateStore.SetPowerLevels(evt.RoomID, evt.Content.GetPowerLevels())
	}
}

type BasicStateStore struct {
	globalLock sync.RWMutex `json:"-"`

	registrationsLock sync.RWMutex                              `json:"-"`
	Registrations     map[string]bool                           `json:"registrations"`
	membershipsLock   sync.RWMutex                              `json:"-"`
	Memberships       map[string]map[string]mautrix.Membership `json:"memberships"`
	powerLevelsLock   sync.RWMutex                              `json:"-"`
	PowerLevels       map[string]*mautrix.PowerLevels          `json:"power_levels"`

	Typing     map[string]map[string]int64 `json:"-"`
	typingLock sync.RWMutex                `json:"-"`
}

func NewBasicStateStore() StateStore {
	return &BasicStateStore{
		Registrations: make(map[string]bool),
		Memberships:   make(map[string]map[string]mautrix.Membership),
		PowerLevels:   make(map[string]*mautrix.PowerLevels),
		Typing:        make(map[string]map[string]int64),
	}
}

func (store *BasicStateStore) RLock() {
	store.registrationsLock.RLock()
	store.membershipsLock.RLock()
	store.powerLevelsLock.RLock()
	store.typingLock.RLock()
}

func (store *BasicStateStore) RUnlock() {
	store.typingLock.RUnlock()
	store.powerLevelsLock.RUnlock()
	store.membershipsLock.RUnlock()
	store.registrationsLock.RUnlock()
}

func (store *BasicStateStore) Lock() {
	store.registrationsLock.Lock()
	store.membershipsLock.Lock()
	store.powerLevelsLock.Lock()
	store.typingLock.Lock()
}

func (store *BasicStateStore) Unlock() {
	store.typingLock.Unlock()
	store.powerLevelsLock.Unlock()
	store.membershipsLock.Unlock()
	store.registrationsLock.Unlock()
}

func (store *BasicStateStore) IsRegistered(userID string) bool {
	store.registrationsLock.RLock()
	defer store.registrationsLock.RUnlock()
	registered, ok := store.Registrations[userID]
	return ok && registered
}

func (store *BasicStateStore) MarkRegistered(userID string) {
	store.registrationsLock.Lock()
	defer store.registrationsLock.Unlock()
	store.Registrations[userID] = true
}

func (store *BasicStateStore) IsTyping(roomID, userID string) bool {
	store.typingLock.RLock()
	defer store.typingLock.RUnlock()
	roomTyping, ok := store.Typing[roomID]
	if !ok {
		return false
	}
	typingEndsAt, _ := roomTyping[userID]
	return typingEndsAt >= time.Now().Unix()
}

func (store *BasicStateStore) SetTyping(roomID, userID string, timeout int64) {
	store.typingLock.Lock()
	defer store.typingLock.Unlock()
	roomTyping, ok := store.Typing[roomID]
	if !ok {
		if timeout >= 0 {
			roomTyping = map[string]int64{
				userID: time.Now().Unix() + timeout,
			}
		} else {
			roomTyping = make(map[string]int64)
		}
	} else {
		if timeout >= 0 {
			roomTyping[userID] = time.Now().Unix() + timeout
		} else {
			delete(roomTyping, userID)
		}
	}
	store.Typing[roomID] = roomTyping
}

func (store *BasicStateStore) GetRoomMemberships(roomID string) map[string]mautrix.Membership {
	store.membershipsLock.RLock()
	memberships, ok := store.Memberships[roomID]
	store.membershipsLock.RUnlock()
	if !ok {
		memberships = make(map[string]mautrix.Membership)
		store.membershipsLock.Lock()
		store.Memberships[roomID] = memberships
		store.membershipsLock.Unlock()
	}
	return memberships
}

func (store *BasicStateStore) GetMembership(roomID, userID string) mautrix.Membership {
	store.membershipsLock.RLock()
	defer store.membershipsLock.RUnlock()
	memberships, ok := store.Memberships[roomID]
	if !ok {
		return mautrix.MembershipLeave
	}
	membership, ok := memberships[userID]
	if !ok {
		return mautrix.MembershipLeave
	}
	return membership
}

func (store *BasicStateStore) IsInRoom(roomID, userID string) bool {
	return store.IsMembership(roomID, userID, "join")
}

func (store *BasicStateStore) IsInvited(roomID, userID string) bool {
	return store.IsMembership(roomID, userID, "join", "invite")
}

func (store *BasicStateStore) IsMembership(roomID, userID string, allowedMemberships ...mautrix.Membership) bool {
	membership := store.GetMembership(roomID, userID)
	for _, allowedMembership := range allowedMemberships {
		if allowedMembership == membership {
			return true
		}
	}
	return false
}

func (store *BasicStateStore) SetMembership(roomID, userID string, membership mautrix.Membership) {
	store.membershipsLock.Lock()
	memberships, ok := store.Memberships[roomID]
	if !ok {
		memberships = map[string]mautrix.Membership{
			userID: membership,
		}
	} else {
		memberships[userID] = membership
	}
	store.Memberships[roomID] = memberships
	store.membershipsLock.Unlock()
}

func (store *BasicStateStore) SetPowerLevels(roomID string, levels *mautrix.PowerLevels) {
	store.powerLevelsLock.Lock()
	store.PowerLevels[roomID] = levels
	store.powerLevelsLock.Unlock()
}

func (store *BasicStateStore) GetPowerLevels(roomID string) (levels *mautrix.PowerLevels) {
	store.powerLevelsLock.RLock()
	levels, _ = store.PowerLevels[roomID]
	store.powerLevelsLock.RUnlock()
	return
}

func (store *BasicStateStore) GetPowerLevel(roomID, userID string) int {
	return store.GetPowerLevels(roomID).GetUserLevel(userID)
}

func (store *BasicStateStore) GetPowerLevelRequirement(roomID string, eventType mautrix.EventType) int {
	return store.GetPowerLevels(roomID).GetEventLevel(eventType)
}

func (store *BasicStateStore) HasPowerLevel(roomID, userID string, eventType mautrix.EventType) bool {
	return store.GetPowerLevel(roomID, userID) >= store.GetPowerLevelRequirement(roomID, eventType)
}
