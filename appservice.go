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
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"

	"maunium.net/go/maulogger"
)

// EventChannelSize is the size for the Events channel in Appservice instances.
var EventChannelSize = 64

// Create a blank appservice instance.
func Create() *Config {
	return &Config{
		LogConfig: CreateLogConfig(),
	}
}

// Load an appservice config from a file.
func Load(path string) (*Config, error) {
	data, readErr := ioutil.ReadFile(path)
	if readErr != nil {
		return nil, readErr
	}

	var config = &Config{}
	yaml.Unmarshal(data, config)
	return config, nil
}

// QueryHandler handles room alias and user ID queries from the homeserver.
type QueryHandler interface {
	QueryAlias(alias string) bool
	QueryUser(userID string) bool
}

// Config is the main config for all appservices.
// It also serves as the appservice instance struct.
type Config struct {
	HomeserverDomain string     `yaml:"homeserver_domain"`
	HomeserverURL    string     `yaml:"homeserver_url"`
	RegistrationPath string     `yaml:"registration"`
	Host             HostConfig `yaml:"host"`
	LogConfig        LogConfig  `yaml:"logging"`

	Registration *Registration     `yaml:"-"`
	Log          *maulogger.Logger `yaml:"-"`

	lastProcessedTransaction string       `yaml:"-"`
	Events                   chan Event   `yaml:"-"`
	QueryHandler             QueryHandler `yaml:"-"`
}

// HostConfig contains info about how to host the appservice.
type HostConfig struct {
	Hostname string `yaml:"hostname"`
	Port     uint16 `yaml:"port"`
	TLSKey   string `yaml:"tls_key,omitempty"`
	TLSCert  string `yaml:"tls_cert,omitempty"`
}

// Address gets the whole address of the Appservice.
func (hc *HostConfig) Address() string {
	return fmt.Sprintf("%s:%d", hc.Hostname, hc.Port)
}

// Save saves this config into a file at the given path.
func (as *Config) Save(path string) error {
	data, err := yaml.Marshal(as)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(path, data, 0644)
}

// YAML returns the config in YAML format.
func (as *Config) YAML() (string, error) {
	data, err := yaml.Marshal(as)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Init initializes the logger and loads the registration of this appservice.
func (as *Config) Init(queryHandler QueryHandler) (bool, error) {
	as.Events = make(chan Event, EventChannelSize)
	as.QueryHandler = queryHandler

	as.Log = maulogger.Create()
	as.LogConfig.Configure(as.Log)
	as.Log.Debugln("Logger initialized successfully.")

	var err error
	as.Registration, err = LoadRegistration(as.RegistrationPath)
	if err != nil {
		return false, err
	}

	as.Log.Debugln("Appservice initialized successfully.")
	return true, nil
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

// CreateLogConfig creates a basic LogConfig.
func CreateLogConfig() LogConfig {
	return LogConfig{
		Directory:       "./logs",
		FileNameFormat:  "%[1]s-%02[2]d.log",
		TimestampFormat: "Jan _2, 2006 15:04:05",
		FileMode:        0600,
		FileDateFormat:  "2006-01-02",
		Debug:           false,
	}
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
	log.FileTimeFormat = lc.FileDateFormat
	log.TimeFormat = lc.TimestampFormat
	if lc.Debug {
		log.PrintLevel = maulogger.LevelDebug.Severity
	}
}
