// matrix-appservice-go - A Matrix application service framework written in Go
// Copyright (C) 2017 Tulir Asokan

// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU General Public License for more details.

// You should have received a copy of the GNU General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package appservice

import (
	"encoding/json"
	"net/http"
)

// EventList contains a list of events.
type EventList struct {
	Events []Event `json:"events"`
}

// Event contains the base fields for events
type Event struct {
	ID               string                 `json:"event_id"`
	Type             string                 `json:"type"`
	Content          map[string]interface{} `json:"content"`
	SenderID         string                 `json:"user_id"`
	RoomID           string                 `json:"room_id"`
	OriginServerTime int64                  `json:"origin_server_ts"`
	Age              int64                  `json:"age"`
}

// WriteBlankOK writes a blank OK message as a reply to a HTTP request.
func WriteBlankOK(w http.ResponseWriter) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("{}"))
}

// Respond responds to a HTTP request with a JSON object.
func Respond(w http.ResponseWriter, data interface{}) error {
	dataStr, err := json.Marshal(data)
	if err != nil {
		return err
	}
	_, err = w.Write([]byte(dataStr))
	return err
}

// Error represents a Matrix protocol error.
type Error struct {
	HTTPStatus int       `json:"-"`
	ErrorCode  ErrorCode `json:"errcode"`
	Message    string    `json:"message"`
}

func (err Error) Write(w http.ResponseWriter) {
	w.WriteHeader(err.HTTPStatus)
	Respond(w, &err)
}

// ErrorCode is the machine-readable code in an Error.
type ErrorCode string

// Native ErrorCodes
const (
	ErrForbidden ErrorCode = "M_FORBIDDEN"
	ErrUnknown   ErrorCode = "M_UNKNOWN"
)

// Custom ErrorCodes
const (
	ErrNoTransactionID ErrorCode = "NET.MAUNIUM.NO_TRANSACTION_ID"
	ErrNoBody          ErrorCode = "NET.MAUNIUM.NO_REUQEST_BODY"
	ErrInvalidJSON     ErrorCode = "NET.MAUNIUM.INVALID_JSON"
)
