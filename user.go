package main

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

type ircUser struct {
	Nick     string      // nickname at the moment.
	User     string      // username
	Host     string      // Userhost
	Modes    string      // Modes currently
	AWAY     bool        // If user is away
	Realname string      // real name
	Writer   chan string // used to write messages to user
	Conn     net.Conn    // pointer to connection
	Server   *Server     // pointer to server
	NickList []string    // Past 5 nicknames - excluding present
}

func (user *ircUser) isValidNick(nick string) bool {
	// nickname   =  ( letter / special ) *8( letter / digit / special / "-" )
	// special = "[", "]", "\", "`", "_", "^", "{", "|", "}"
	re_special := "`" + `\[\]\_^{|}`
	isValid, _ := regexp.MatchString(`^[(A-Za-z)(`+re_special+`)][A-Za-z0-9`+re_special+"]{0,8}$", nick)
	return isValid
}

func (user *ircUser) getHostAddr() (h string) {
	h, _, _ = net.SplitHostPort(user.Conn.LocalAddr().String()) // Return IP/Host
	return
}

func (user *ircUser) updateNick(nick string) {
	// More focused on nickname changes.
	if _, ok := user.Server.Clients[user.Nick]; !ok {
		user.Nick = nick
		user.Server.Unregistered[&user.Conn] = user
	} else {
		delete(user.Server.Clients, user.Nick)
		user.Nick = nick
		user.updateUser()
	}
}

func (user *ircUser) updateUser() {
	// Set host manually - In case provided pointer doesn't have set.
	user.Host = user.Nick + "!~" + user.User + "@" + user.getHostAddr()
	user.Server.Clients[user.Nick] = user
	delete(user.Server.Unregistered, &user.Conn)
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
