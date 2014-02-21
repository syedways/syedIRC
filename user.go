package main

import (
	"fmt"
	"net"
	"regexp"
	"strings"
	"sync"
)

type ircUser struct {
	Nick     string      // nickname at the moment.
	User     string      // username
	Host     string      // Userhost
	Mode     int         // Modes(converted to ints, not sure about this yet.)
	AWAY     bool        // If user is away
	Realname string      // real name
	Writer   chan string // used to write messages to user
	Conn     net.Conn    // pointer to connection
	Server   Server      // pointer to server
	NickList []byte      // Past 5 nicknames - excluding present
	mu       sync.Mutex  // Sync it up
}

func isValidNick(nick string) bool {
	// nickname   =  ( letter / special ) *8( letter / digit / special / "-" )
	re_between := "`_\\^\\{\\|\\}][A-Za-z0-9\\[\\]\\`"
	re_nick := `[A-Za-z\[\]\\` + re_between + `_\^\{\|\}\-]{0,8}`
	isValid, _ := regexp.MatchString(re_nick, nick)
	return isValid
}
func (user *ircUser) getHostAddr() (h string) {
	h, _, _ = net.SplitHostPort(user.Conn.LocalAddr().String()) // Return IP/Host
	return
}

func (user *ircUser) updateUser() {
	user.mu.Lock()
	if user.Nick != "" && user.User != "" {
		h, _, _ := net.SplitHostPort(user.Conn.LocalAddr().String()) // Return IP/Host
		user.Host = user.Nick + "!~" + user.User + "@" + h
	}
	if user.Nick != "AUTH" && user.User == "" {
		user.Server.Unregistered[&user.Conn] = user
	}
	if _, ok := user.Server.Unregistered[&user.Conn]; ok && user.User != "" {
		user.Server.Clients[user.Nick] = user
		delete(user.Server.Unregistered, &user.Conn)
	}
	user.mu.Unlock()
}

func (user *ircUser) getUser() (u *ircUser) {
	user.mu.Lock()
	u = user
	// Get user from unregistered map
	if _, ok := user.Server.Unregistered[&user.Conn]; ok {
		u = user.Server.Unregistered[&user.Conn]
	}
	// Get user from clients map
	if _, ok := user.Server.Clients[user.Nick]; ok {
		u = user.Server.Clients[user.Nick]
	}
	user.mu.Unlock()
	return
}

func (user *ircUser) deleteUser() {
	delete(user.Server.Unregistered, &user.Conn)
	delete(user.Server.Clients, user.Nick)
}

func (user *ircUser) Command(command string, line string) {
	out := fmt.Sprintf(":%s %s %s :%s", user.Nick, command, user.Nick, line)
	user.Writer <- out
}

func (user *ircUser) serverWrite(variable string, command string, line string) {
	out := fmt.Sprintf(":%s %s %s :%s", user.Server.Host, command, variable, line)
	user.Writer <- out
}

func (user *ircUser) sendNumeric(numeric string, args ...string) {
	out := fmt.Sprintf(":%s %s %s %s", user.Server.Host, numeric, user.Nick, strings.Join(args, " "))
	user.Writer <- out
}

func (user *ircUser) raw(line ...string) {
	out := fmt.Sprintf(strings.Join(line, " "))
	user.Writer <- out
}
