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
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

func readString(reader *bufio.Reader, message, defaultValue string) (string, error) {
	color.Green(message)
	if len(defaultValue) > 0 {
		fmt.Printf("[%s]", defaultValue)
	}
	fmt.Print("> ")
	val, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	val = strings.TrimSuffix(val, "\n")
	if len(val) == 0 {
		return defaultValue
	}
	return val, nil
}

const (
	yes      = "yes"
	yesShort = "y"
)

// GenerateRegistration asks the user questions and generates a config and registration based on the answers.
func GenerateRegistration(asName, botName string, reserveRooms, reserveUsers bool) {
	var boldCyan = color.New(color.FgCyan).Add(color.Bold)
	var boldGreen = color.New(color.FgGreen).Add(color.Bold)
	boldCyan.Println("Generating appservice config and registration.")
	reader := bufio.NewReader(os.Stdin)

	name := readString(reader, "Enter name for appservice", asName)
	registration := CreateRegistration(name)
	config := Create()
	registration.RateLimited = false

	registration.SenderLocalpart = readString(reader, "Enter bot username", botName)

	asProtocol := readString(reader, "Enter appservice host protocol", "http")
	if asProtocol == "https" {
		wantSSL := strings.ToLower(readString(reader, "Do you want the appservice to handle SSL [yes/no]?", "yes"))
		if wantSSL == yes {
			config.Host.TLSCert = readString(reader, "Enter path to SSL certificate", "appservice.crt")
			config.Host.TLSKey = readString(reader, "Enter path to SSL key", "appservice.key")
		}
	}
	asHostname := readString(reader, "Enter appservice hostname", "localhost")
	asPort, convErr := strconv.Atoi(readString(reader, "Enter appservice host port", "29313"))
	if convErr != nil {
		fmt.Println("Failed to parse port:", err)
		return
	}
	registration.URL = fmt.Sprintf("%s://%s:%d", asProtocol, asHostname, asPort)
	config.Host.Hostname = asHostname
	config.Host.Port = uint16(asPort)

	config.HomeserverURL = readString(reader, "Enter homeserver address", "http://localhost:8008")
	config.HomeserverDomain = readString(reader, "Enter homeserver domain", "example.com")
	config.LogConfig.Directory = readString(reader, "Enter directory for logs", "./logs")
	os.MkdirAll(config.LogConfig.Directory, 0755)

	if reserveRooms || reserveUsers {
		for {
			namespace := readString(reader, "Enter namespace prefix", fmt.Sprintf("_%s_", name))
			roomNamespaceRegex, err := regexp.Compile(fmt.Sprintf("#%s.+:%s", namespace, config.HomeserverDomain))
			if err != nil {
				fmt.Println(err)
				continue
			}
			userNamespaceRegex, regexpErr := regexp.Compile(fmt.Sprintf("@%s.+:%s", namespace, config.HomeserverDomain))
			if regexpErr != nil {
				fmt.Println("Failed to generate regexp for the userNamespace:", err)
				return
			}
			if reserveRooms {
				registration.Namespaces.RegisterRoomAliases(roomNamespaceRegex, true)
			}
			if reserveUsers {
				registration.Namespaces.RegisterUserIDs(userNamespaceRegex, true)
			}
			break
		}
	}

	boldCyan.Println("\n==== Registration generated ====")
	color.Yellow(registration.YAML())

	ok := strings.ToLower(readString(reader, "Does the registration look OK [yes/no]?", "yes"))
	if ok != yesShort && ok != yes {
		fmt.Println("Cancelling generation.")
		return
	}

	path := readString(reader, "Where should the registration be saved?", "registration.yaml")
	err := registration.Save(path)
	if err != nil {
		fmt.Println("Failed to save registration:", err)
		return
	}
	boldGreen.Println("Registration saved.")

	config.RegistrationPath = path

	boldCyan.Println("\n======= Config generated =======")
	color.Yellow(config.YAML())

	ok = strings.ToLower(readString(reader, "Does the config look OK [yes/no]?", "yes"))
	if ok != yesShort && ok != yes {
		fmt.Println("Cancelling generation.")
		return
	}

	path = readString(reader, "Where should the config be saved?", "config.yaml")
	err = config.Save(path)
	if err != nil {
		fmt.Println("Failed to save config:", err)
		return
	}
	boldGreen.Println("Config saved.")
}
