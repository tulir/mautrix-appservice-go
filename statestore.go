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
	GetMember(roomID, userID string) mautrix.Member
	TryGetMember(roomID, userID string) (mautrix.Member, bool)
	SetMembership(roomID, userID string, membership mautrix.Membership)
	SetMember(roomID, userID string, member mautrix.Member)

	SetPowerLevels(roomID string, levels *mautrix.PowerLevels)
	GetPowerLevels(roomID string) *mautrix.PowerLevels
	GetPowerLevel(roomID, userID string) int
	GetPowerLevelRequirement(roomID string, eventType mautrix.EventType) int
	HasPowerLevel(roomID, userID string, eventType mautrix.EventType) bool
}

func (as *AppService) UpdateState(evt *mautrix.Event) {
	switch evt.Type {
	case mautrix.StateMember:
		as.StateStore.SetMember(evt.RoomID, evt.GetStateKey(), evt.Content.Member)
	case mautrix.StatePowerLevels:
		as.StateStore.SetPowerLevels(evt.RoomID, evt.Content.GetPowerLevels())
	}
}

type TypingStateStore struct {
	typing     map[string]map[string]int64
	typingLock sync.RWMutex
}

func NewTypingStateStore() *TypingStateStore {
	return &TypingStateStore{
		typing: make(map[string]map[string]int64),
	}
}

func (store *TypingStateStore) IsTyping(roomID, userID string) bool {
	store.typingLock.RLock()
	defer store.typingLock.RUnlock()
	roomTyping, ok := store.typing[roomID]
	if !ok {
		return false
	}
	typingEndsAt, _ := roomTyping[userID]
	return typingEndsAt >= time.Now().Unix()
}

func (store *TypingStateStore) SetTyping(roomID, userID string, timeout int64) {
	store.typingLock.Lock()
	defer store.typingLock.Unlock()
	roomTyping, ok := store.typing[roomID]
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
	store.typing[roomID] = roomTyping
}

type BasicStateStore struct {
	registrationsLock sync.RWMutex                         `json:"-"`
	Registrations     map[string]bool                      `json:"registrations"`
	membersLock       sync.RWMutex                         `json:"-"`
	Members           map[string]map[string]mautrix.Member `json:"memberships"`
	powerLevelsLock   sync.RWMutex                         `json:"-"`
	PowerLevels       map[string]*mautrix.PowerLevels      `json:"power_levels"`

	*TypingStateStore
}

func NewBasicStateStore() StateStore {
	return &BasicStateStore{
		Registrations:    make(map[string]bool),
		Members:          make(map[string]map[string]mautrix.Member),
		PowerLevels:      make(map[string]*mautrix.PowerLevels),
		TypingStateStore: NewTypingStateStore(),
	}
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

func (store *BasicStateStore) GetRoomMembers(roomID string) map[string]mautrix.Member {
	store.membersLock.RLock()
	members, ok := store.Members[roomID]
	store.membersLock.RUnlock()
	if !ok {
		members = make(map[string]mautrix.Member)
		store.membersLock.Lock()
		store.Members[roomID] = members
		store.membersLock.Unlock()
	}
	return members
}

func (store *BasicStateStore) GetMembership(roomID, userID string) mautrix.Membership {
	return store.GetMember(roomID, userID).Membership
}

func (store *BasicStateStore) GetMember(roomID, userID string) mautrix.Member {
	member, ok := store.TryGetMember(roomID, userID)
	if !ok {
		member.Membership = mautrix.MembershipLeave
	}
	return member
}

func (store *BasicStateStore) TryGetMember(roomID, userID string) (member mautrix.Member, ok bool) {
	store.membersLock.RLock()
	defer store.membersLock.RUnlock()
	members, membersOk := store.Members[roomID]
	if !membersOk {
		return
	}
	member, ok = members[userID]
	return
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
	store.membersLock.Lock()
	members, ok := store.Members[roomID]
	if !ok {
		members = map[string]mautrix.Member{
			userID: {Membership: membership},
		}
	} else {
		member, ok := members[userID]
		if !ok {
			members[userID] = mautrix.Member{Membership: membership}
		} else {
			member.Membership = membership
			members[userID] = member
		}
	}
	store.Members[roomID] = members
	store.membersLock.Unlock()
}

func (store *BasicStateStore) SetMember(roomID, userID string, member mautrix.Member) {
	store.membersLock.Lock()
	members, ok := store.Members[roomID]
	if !ok {
		members = map[string]mautrix.Member{
			userID: member,
		}
	} else {
		members[userID] = member
	}
	store.Members[roomID] = members
	store.membersLock.Unlock()
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
