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
	"fmt"
	"os"

	"maunium.net/go/maulogger"
)

// EventChannelSize is the size for the Events channel in Appservice instances.
var EventChannelSize = 64

// Config is the main config for all appservices.
// It also serves as the appservice instance struct.
type Config struct {
	HomeserverDomain string    `yaml:"homeserver_domain"`
	HomeserverURL    string    `yaml:"homeserver_url"`
	RegistrationPath string    `yaml:"registration"`
	Port             uint8     `yaml:"listen_port"`
	LogConfig        LogConfig `yaml:"logging"`

	Registration *Registration     `yaml:"-"`
	Log          *maulogger.Logger `yaml:"-"`

	lastProcessedTransaction string                     `yaml:"-"`
	Events                   chan Event                 `yaml:"-"`
	EventListeners           map[string][]EventListener `yaml:"-"`
}

// AddEventListener adds an event listener to this appservice.
func (as *Config) AddEventListener(event string, listener EventListener) {
	arr := as.EventListeners[event]
	if arr == nil {
		arr = []EventListener{listener}
	} else {
		arr = append(arr, listener)
	}
	as.EventListeners[event] = arr
}

// Init initializes the logger and loads the registration of this appservice.
func (as *Config) Init() bool {
	as.Events = make(chan Event, EventChannelSize)
	as.EventListeners = make(map[string][]EventListener)

	as.Log = maulogger.Create()
	as.LogConfig.Configure(as.Log)
	as.Log.Debugln("Logger initialized successfully.")

	var err error
	as.Registration, err = LoadRegistration(as.RegistrationPath)
	if err != nil {
		as.Log.Fatalln("Failed to load registration:", err)
		return false
	}

	as.Log.Debugln("Appservice initialized successfully.")
	return true
}

// LogConfig contains configs for the logger.
type LogConfig struct {
	Directory       string `yaml:"directory"`
	FileNameFormat  string `yaml:"file_name_format"`
	FileDateFormat  string `yaml:"file_date_format"`
	FileMode        uint32 `yaml:"file_mode"`
	TimestampFormat string `yaml:"timestamp_format"`
	Debug           bool   `yaml:"print_debug"`
}

// GetFileFormat returns a mauLogger-compatible logger file format based on the data in the struct.
func (lc LogConfig) GetFileFormat() maulogger.LoggerFileFormat {
	path := lc.FileNameFormat
	if len(lc.Directory) > 0 {
		path = lc.Directory + "/" + path
	}

	return func(now string, i int) string {
		return fmt.Sprintf(path, now, i)
	}
}

// Configure configures a mauLogger instance with the data in this struct.
func (lc LogConfig) Configure(log *maulogger.Logger) {
	log.FileFormat = lc.GetFileFormat()
	log.FileMode = os.FileMode(lc.FileMode)
	log.TimeFormat = lc.TimestampFormat
	if lc.Debug {
		log.PrintLevel = maulogger.LevelDebug.Severity
	}
}
