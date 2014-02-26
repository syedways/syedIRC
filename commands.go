package main

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

func IRC_USER(msg *ircMessage) (string, string) {
	// USER <username> <mode> * <:Real name>
	msg.User.User = msg.Payload[0]
	msg.User.Mode, _ = strconv.Atoi(msg.Payload[1])
	msg.User.Realname = msg.Payload[3][1:]

	if _, ok := msg.Server.Clients[msg.User.Nick]; !ok {
		msg.User.updateUser() // Register User

		go func() {
			user := msg.User
			// If malformed nick, we will wait for manual mode update from client.
			if user.Nick != "AUTH" {
				defer user.Command("MODE", "+i")
			}

			// Send initial notices. In the future will actually check for hostname and ident
			user.serverWrite(user.Nick, "NOTICE", "*** Looking up your hostname...")
			user.serverWrite(user.Nick, "NOTICE", "*** Checking Ident")
			user.serverWrite(user.Nick, "NOTICE", "*** Found your hostname")
			time.Sleep(5 * 1e9) // Artificial wait - Allows us a nice period to fix any malformed nicks.
			user.serverWrite(user.Nick, "NOTICE", "*** No Ident response")

			// WELCOME messages
			user.sendNumeric(RPL_WELCOME, ":Welcome to the "+user.Server.Name+" Internet Relay Chat Network "+
				user.Host)
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
		}()
	} else {
		return ERR_ALREADYREGISTERED, ":You may not reregister"
	}
	return "", ""
}

func IRC_NICK(msg *ircMessage) (string, string) {
	// NICK <nickname>
	inputNick := msg.Payload[0]

	// If nickname is not valid.
	if inputNick == "AUTH" || !msg.User.isValidNick(inputNick) {
		return ERR_ERRONEUSNICKNAME, msg.Payload[0] + " :Erroneous Nickname."
	}
	// If nickname exists.
	if e, _, u := msg.Server.nickExists(inputNick); e {
		return ERR_NICKNAMEINUSE, u.Nick + " :This nickname is already in use."
	}

	if _, ok := msg.Server.Clients[msg.User.Nick]; ok {
		// If registered - Notify client nick change was successful
		msg.User.raw(":"+msg.User.Host, "NICK", ":"+inputNick)
	}
	msg.User.updateNick(inputNick)
	return "", ""
}

func IRC_CAP(msg *ircMessage) (string, string) {
	// CAP LS
	return "", ""
}

func IRC_MODE(msg *ircMessage) (string, string) {
	// MODE <nick> +/-<mode>
	if msg.Payload[0] == msg.User.Nick {
		// Will add regex for mode later.
		msg.User.Command("MODE", msg.Payload[1])
	} else {
		return ERR_USERSDONTMATCH, msg.User.Nick + " :You can not change modes for other users."
	}
	return "", ""
}

func IRC_PONG(msg *ircMessage) (string, string) {
	// PING :<payload>
	msg.User.serverWrite(msg.User.Server.Host, "PONG", msg.Payload[0])
	return "", ""
}

func IRC_USERHOST(msg *ircMessage) (string, string) {
	// USERHOST :<nick> <nick> <nick> <nick> <nick>
	response := []string{} // Create a response array.
	// Only works for 5 nicknames.
	iter := msg.Payload
	if len(msg.Payload) >= 5 {
		iter = msg.Payload[0:5]
	}
	for _, nick := range iter {
		// nickname=+(-)userid@host
		if e, r, u := msg.Server.nickExists(nick); e && r { // If user exists and is registered.
			user := []string{u.Nick + "=", "+", u.User + "@", u.getHostAddr()}
			if u.AWAY {
				user[1] = "-"
			}
			response = append(response, strings.Join(user, ""))
		}
	}
	return RPL_USERHOST, ":" + strings.Join(response, " ")
}

func IRC_ISON(msg *ircMessage) (string, string) {
	// ISON :<nick>...
	response := []string{}
	for _, nick := range msg.Payload {
		if e, _, u := msg.Server.nickExists(nick); e {
			response = append(response, u.Nick)
		}
	}
	return RPL_ISON, ":" + strings.Join(response, " ")
}

func IRC_QUIT(msg *ircMessage) (string, string) {
	// QUIT
	msg.User.deleteUser()
	fmt.Printf("We've dropped connection to: %s, he has left the building.\n", msg.User.Nick)
	return "", ""
}
