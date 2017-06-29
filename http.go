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
	"io/ioutil"
	"net/http"

	"github.com/gorilla/mux"
)

// Listen starts the HTTP server that listens for calls from the Matrix homeserver.
func (as *Config) Listen() {
	r := mux.NewRouter()
	r.HandleFunc("/transactions/{txnID}", as.PutTransaction).Methods(http.MethodPut)
	http.Handle("/", r)
}

// CheckServerToken checks if the given request originated from the Matrix homeserver.
func (as *Config) CheckServerToken(w http.ResponseWriter, r *http.Request) bool {
	query := r.URL.Query()
	val, ok := query["access_token"]
	if !ok {
		Error{
			ErrorCode:  ErrForbidden,
			HTTPStatus: http.StatusForbidden,
			Message:    "Bad token supplied.",
		}.Write(w)
		return false
	}
	for _, str := range val {
		return str == as.Registration.ServerToken
	}
	return false
}

// PutTransaction handles a /transactions PUT call from the homeserver.
func (as *Config) PutTransaction(w http.ResponseWriter, r *http.Request) {
	if !as.CheckServerToken(w, r) {
		return
	}

	vars := mux.Vars(r)
	txnID := vars["txnID"]
	if len(txnID) == 0 {
		Error{
			ErrorCode:  ErrNoTransactionID,
			HTTPStatus: http.StatusBadRequest,
			Message:    "Missing transaction ID.",
		}.Write(w)
		return
	}
	defer r.Body.Close()
	body, err := ioutil.ReadAll(r.Body)
	if err != nil || len(body) == 0 {
		Error{
			ErrorCode:  ErrNoBody,
			HTTPStatus: http.StatusBadRequest,
			Message:    "Missing request body.",
		}.Write(w)
		return
	}
	if as.lastProcessedTransaction == txnID {
		// Duplicate transaction ID: no-op
		WriteBlankOK(w)
		return
	}

	eventList := EventList{}
	err = json.Unmarshal(body, &eventList)
	if err != nil {
		Error{
			ErrorCode:  ErrInvalidJSON,
			HTTPStatus: http.StatusBadRequest,
			Message:    "Failed to parse body JSON.",
		}.Write(w)
		return
	}

	for _, event := range eventList.Events {
		as.Log.Debugln("Received event", event.ID)
		as.Events <- event
		listeners, ok := as.EventListeners[event.Type]
		if ok {
			for _, listener := range listeners {
				listener(event)
			}
		}
	}
	as.lastProcessedTransaction = txnID
	WriteBlankOK(w)
}
