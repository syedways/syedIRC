package main

import (
	"fmt"
	"strconv"
)

func IRC_USER(msg *ircMessage) {
	// USER <username> <mode> * <:Real name>
	// We get the username, and realname and send it to the channel to update.
	msg.User.User = msg.Payload[0]
	msg.User.Mode, _ = strconv.Atoi(msg.Payload[1])
	msg.User.Realname = msg.Payload[3]

	// Add to fully registered map.
	if msg.User.Nick != "AUTH" && msg.User.User != "" && msg.User.Realname != "" {
		msg.User.updateUser()
		msg.User.welcome_message()
	} else {
		fmt.Printf("USER command failed, ending user connection.")
		msg.User.deleteUser()
		msg.User.Conn.Close()
	}
}

func IRC_NICK(msg *ircMessage) {
	// NICK <nickname>
	if msg.Payload[0] == "AUTH" { // also need to add checks if nickname fits rfc.
		msg.User.sendNumeric(ERR_ERRONEUSNICKNAME, msg.Payload[0]+" :Erroneous Nickname.")
		return
	}

	// Check if nickname is in use. - FIX
	for _, v := range msg.User.Server.Clients {
		if v.Nick == msg.Payload[0] {
			msg.User.sendNumeric(ERR_NICKNAMEINUSE, msg.Payload[0]+" :This nickname is already in use.")
			return
		}
	}

	// Check if current nickname is registered, if not register it. - FIX
	if _, ok := msg.Server.Clients[&msg.User.Conn]; ok {
		msg.User.raw(":"+msg.User.getFullHost(), "NICK", ":"+msg.Payload[0])
		msg.User.Nick = msg.Payload[0]
		msg.User.updateUser() // Add new user.
	} else {
		msg.User.Nick = msg.Payload[0]
		msg.User.updateUser()
	}
}

func IRC_CAP(msg *ircMessage) {
	// CAP LS
	return
}

func IRC_MODE(msg *ircMessage) {
	// MODE <nick> +/-<mode>
	return
}

func IRC_PONG(msg *ircMessage) {
	// PING :<payload>
	msg.User.serverWrite(msg.User.Server.Host, "PONG", msg.Payload[0])
}

func IRC_QUIT(msg *ircMessage) {
	// QUIT
	msg.User.deleteUser()
	fmt.Printf("We've dropped connection to: %s, he has left the building.\n", msg.User.Nick)
}
