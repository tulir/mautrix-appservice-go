// Copyright (c) 2019 Tulir Asokan
//
// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/.

package appservice

import (
	"encoding/json"
	"fmt"
	"runtime/debug"
	"time"

	"maunium.net/go/mautrix"
)

type Syncer struct {
	*AppService
}

func (as *Syncer) OnFailedSync(res *mautrix.RespSync, err error) (time.Duration, error) {
	as.Log.Errorln("Sync errored:", err)
	return 10 * time.Second, nil
}

func (as *Syncer) ProcessResponse(resp *mautrix.RespSync, since string) (err error) {
	if since == "" {
		return nil
	}
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("ProcessResponse panicked! since=%s panic=%s\n%s", since, r, debug.Stack())
		}
	}()

	for roomID, roomData := range resp.Rooms.Join {
		for _, evt := range roomData.State.Events {
			evt.RoomID = roomID
			as.UpdateState(evt)
			as.Events <- evt
		}
		for _, evt := range roomData.Timeline.Events {
			evt.RoomID = roomID
			as.Events <- evt
		}
	}
	for roomID, roomData := range resp.Rooms.Invite {
		for _, evt := range roomData.State.Events {
			evt.RoomID = roomID
			as.UpdateState(evt)
			as.Events <- evt
		}
	}
	for roomID, roomData := range resp.Rooms.Leave {
		for _, evt := range roomData.Timeline.Events {
			if evt.StateKey != nil {
				evt.RoomID = roomID
				as.UpdateState(evt)
				as.Events <- evt
			}
		}
	}
	return
}

func (as *Syncer) GetFilterJSON(_ string) json.RawMessage {
	return json.RawMessage(`{"room":{"timeline":{"limit":50}}}`)
}

type Store struct {
	*AppService
}

func (as *Store) SaveFilterID(_, filterID string) {
	as.Sync.FilterID = filterID
}

func (as *Store) LoadFilterID(_ string) string {
	return as.Sync.FilterID
}

func (as *Store) SaveNextBatch(_, nextBatch string) {
	as.Sync.NextBatch = nextBatch
}

func (as *Store) LoadNextBatch(_ string) string {
	return as.Sync.NextBatch
}

func (as *Store) SaveRoom(_ *mautrix.Room) {}

func (as *Store) LoadRoom(roomID string) *mautrix.Room {
	return nil
}

func (as *AppService) startSync() {
	client := as.BotClient()
	client.Syncer = &Syncer{as}
	client.Store = &Store{as}
	as.Log.Infoln("Starting syncing")
	err := client.Sync()
	if err != nil {
		as.Log.Errorln("Sync returned error:", err)
	}
}

func (as *AppService) stopSync() {
	as.BotClient().StopSync()
}
