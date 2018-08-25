package appservice

import (
	"maunium.net/go/gomatrix"
	"strings"
)

type StateStore interface {
	IsRegistered(userID string) bool
	MarkRegistered(userID string)

	IsInRoom(roomID, userID string) bool
	SetMembership(roomID, userID, membership string)

	SetPowerLevels(roomID string, levels gomatrix.PowerLevels)
	GetPowerLevels(roomID string) gomatrix.PowerLevels
	GetPowerLevel(roomID, userID string) int
	GetPowerLevelRequirement(roomID, eventType string, isState bool) int
	HasPowerLevel(roomID, userID, eventType string, isState bool) bool
}

func (as *AppService) UpdateState(evt *gomatrix.Event) {
	switch evt.Type {
	case gomatrix.StateMember:
		as.StateStore.SetMembership(evt.RoomID, evt.GetStateKey(), evt.Content.Membership)
	}
}

type BasicStateStore struct {
	Registrations map[string]bool                 `json:"registrations"`
	Memberships   map[string]map[string]string    `json:"memberships"`
	PowerLevels   map[string]gomatrix.PowerLevels `json:"power_levels"`
}

func NewBasicStateStore() *BasicStateStore {
	return &BasicStateStore{
		Registrations: make(map[string]bool),
		Memberships:   make(map[string]map[string]string),
		PowerLevels:   make(map[string]gomatrix.PowerLevels),
	}
}

func (store BasicStateStore) IsRegistered(userID string) bool {
	registered, ok := store.Registrations[userID]
	return ok && registered
}

func (store BasicStateStore) MarkRegistered(userID string) {
	store.Registrations[userID] = true
}

func (store BasicStateStore) GetRoomMemberships(roomID string) map[string]string {
	memberships, ok := store.Memberships[roomID]
	if !ok {
		memberships = make(map[string]string)
		store.Memberships[roomID] = memberships
	}
	return memberships
}

func (store BasicStateStore) GetMembership(roomID, userID string) string {
	membership, ok := store.GetRoomMemberships(roomID)[userID]
	if !ok {
		return "leave"
	}
	return membership
}

func (store BasicStateStore) IsInRoom(roomID, userID string) bool {
	return store.GetMembership(roomID, userID) == "join"
}

func (store BasicStateStore) SetMembership(roomID, userID, membership string) {
	store.GetRoomMemberships(roomID)[userID] = strings.ToLower(membership)
}

func (store BasicStateStore) SetPowerLevels(roomID string, levels gomatrix.PowerLevels) {
	store.PowerLevels[roomID] = levels
}

func (store BasicStateStore) GetPowerLevels(roomID string) (levels gomatrix.PowerLevels) {
	levels, _ = store.PowerLevels[roomID]
	return
}

func (store BasicStateStore) GetPowerLevel(roomID, userID string) int {
	levels := store.GetPowerLevels(roomID)
	userLevel, ok := levels.Users[userID]
	if !ok {
		return levels.UsersDefault
	}
	return userLevel
}

func (store BasicStateStore) GetPowerLevelRequirement(roomID, eventType string, isState bool) int {
	levels := store.GetPowerLevels(roomID)
	eventLevel, ok := levels.Events[eventType]
	if ok {
		return eventLevel
	}
	switch eventType {
	case "kick":
		return levels.Kick()
	case "invite":
		return levels.Invite()
	case "redact":
		return levels.Redact()
	}
	if isState {
		return levels.StateDefault()
	}
	return levels.EventsDefault
}

func (store BasicStateStore) HasPowerLevel(roomID, userID, eventType string, isState bool) bool {
	return store.GetPowerLevel(roomID, userID) >= store.GetPowerLevelRequirement(roomID, eventType, isState)
}
