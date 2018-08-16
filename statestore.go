package appservice

import (
	"strings"
)

type StateStore interface {
	IsRegistered(userID string) bool
	MarkRegistered(userID string)

	IsInRoom(userID, roomID string) bool
	SetMembership(userID, roomID, membership string)
}

type BasicStateStore struct {
	Registrations map[string]bool              `json:"registrations"`
	Memberships   map[string]map[string]string `json:"memberships"`
}

func NewBasicStateStore() *BasicStateStore {
	return &BasicStateStore{
		Registrations: make(map[string]bool),
		Memberships:   make(map[string]map[string]string),
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

func (store BasicStateStore) IsInRoom(userID, roomID string) bool {
	return store.GetMembership(roomID, userID) == "join"
}

func (store BasicStateStore) SetMembership(roomID, userID, membership string) {
	store.GetRoomMemberships(roomID)[userID] = strings.ToLower(membership)
}
