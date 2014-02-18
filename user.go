package main

import (
	"fmt"
	"net"
	"strings"
	"sync"
)

type ircUser struct {
	Nick     string      // nickname at the moment.
	User     string      // username
	Host     string      // Host
	Mode     int         // Modes(converted to ints, not sure about this yet.)
	Realname string      // real name
	Writer   chan string // used to write messages to user
	Conn     net.Conn    // pointer to connection
	Server   Server      // pointer to server
	mu       sync.Mutex  // Sync it up
}

func (user *ircUser) updateUser() {
	user.mu.Lock()
	user.Server.Clients[&user.Conn] = user
	user.mu.Unlock()
}

func (user *ircUser) getUser() (u *ircUser) {
	user.mu.Lock()
	u = user
	// Get user from clients map
	if _, ok := user.Server.Clients[&user.Conn]; ok {
		u = user.Server.Clients[&user.Conn]
	}
	user.mu.Unlock()
	return
}

func (user *ircUser) deleteUser() {
	delete(user.Server.Clients, &user.Conn)
}

func (user *ircUser) getFullHost() (host string) {
	// Might need changing later, to add identd support.
	h, _, _ := net.SplitHostPort(user.Host) // Get IP/Host
	host = user.Nick + "!~" + user.User + "@" + h
	return
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

func (user *ircUser) welcome_message() {
	// Send initial notices. In the future will actually check for hostname and ident
	user.serverWrite(user.Nick, "NOTICE", "*** Looking up your hostname...")
	user.serverWrite(user.Nick, "NOTICE", "*** Checking Ident")
	user.serverWrite(user.Nick, "NOTICE", "*** Found your hostname")
	user.serverWrite(user.Nick, "NOTICE", "*** No Ident response")

	// WELCOME messages
	user.sendNumeric(RPL_WELCOME, ":Welcome to the "+user.Server.Name+" Internet Relay Chat Network "+
		user.getFullHost())
	user.sendNumeric(RPL_YOURHOST, ":Your host is "+user.Server.Host+", running goIRC v1.0.0")
	user.sendNumeric(RPL_CREATED, ":This server was created Tue Dec 17 2013 at 23:43:26 EST") // Needs to be non-hardcoded
	user.sendNumeric(RPL_SERVERVERSION, ":"+user.Server.Host+" goIRC.0.0 iowghraAsORTVSxNCWqBzvdHtGpfF lvhopsmntikrRcaqOALQbSeIKVfMCuzNTGjHFEB")
	user.sendNumeric(RPL_ISUPPORT, ":CHANTYPES=#")
	user.sendNumeric(RPL_ISUPPORT, ":CHANMODES= BLAH BLAH BLAH")
	user.sendNumeric(RPL_ISUPPORT, ":PREFIX=(BLAH BLAH BLAH)")
	user.sendNumeric(RPL_ISUPPORT, ":are supported by this server")
	user.sendNumeric(RPL_MOTDSTART, ":"+user.Server.Host+" Message of the Day -")
	user.sendNumeric(RPL_MOTD, ":- Trickle down economics is a sham. - Richard 'two-buck chuck' Holland")
	user.sendNumeric(RPL_ENDOFMOTD, ":End of /MOTD")
	user.Command("MODE", "+i") // Needs to be changed in the future.
}
