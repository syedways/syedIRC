package main

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

func IRC_USER(msg *ircMessage) (string, string) {
	// USER <username> <mode> * <:Real name>
	msg.User.User = msg.Payload[0]
	msg.User.Realname = msg.Payload[3][1:]

	if _, ok := msg.Server.Clients[msg.User.Nick]; !ok {
		msg.User.updateUser() // Register User

		go func() {
			user := msg.User
			// If malformed nick, we will wait for manual mode update from client.
			if user.Nick != "AUTH" {
				msg.User.Modes = "i"
				defer user.Command("MODE", "+i")
				defer fmt.Println("Sent welcome messages and MOTD to:", msg.User.Nick)
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
	// a - user is flagged as away; // can't be set with this command
	// i - marks a users as invisible;
	// w - user receives wallops;
	// r - restricted user connection; // obosolete in this implementation
	// o - operator flag; // not implemented; when it is, only unset is allowed
	// O - local operator flag; // not implemented; when it is, only unset is allowed
	// s - marks a user for receipt of server notices. // obsolete.

	usableModes := "iw"            // Usable modes.
	setModes, unsetModes := "", "" // Sent to client at end.
	unknownReached := false        // Reached an unknown mode, return an error.

	if strings.ToLower(msg.Payload[0]) != strings.ToLower(msg.User.Nick) {
		return ERR_USERSDONTMATCH, msg.User.Nick + " :Cannot change mode for other users"
	}

	// If only provided nick, return modes of self.
	if len(msg.Payload) == 1 {
		return RPL_UMODEIS, "+" + msg.User.Modes
	}

	// Extract all mode changes from message
	r, _ := regexp.Compile(`(-|\+*)([A-Za-z]+)`)
	chmodes := r.FindAllStringSubmatch(msg.Payload[1], -1)

	for _, d := range chmodes {
		// chmodes is a list of seperated mode changes
		// -wi+iw-iw+iw-w+w would result in 6 iterations
		chType := true // true = set mode, false = unset mode:
		if d[1] == "-" {
			chType = false
		}

		// Sort through modes for this iteration
		for _, char := range d[2] {
			char := string(char)

			// See if mode is usable, if not dump and send err message at the end.
			if !strings.Contains(usableModes, char) {
				unknownReached = true
				continue
			}
			userHasMode := strings.Contains(msg.User.Modes, char)

			// Unset a mode, check if already in the list to be unset
			if !chType && !strings.Contains(unsetModes, char) {
				// if mode exists in setmodes, or user doesn't have mode.
				if strings.Contains(setModes, char) || !userHasMode {
					setModes = strings.Replace(setModes, char, "", -1)
					continue
				}
				unsetModes = unsetModes + char
			}

			// Set a mode, check if already in the list to be set
			if chType && !strings.Contains(setModes, char) {
				// if mode exists in unsetmodes, or user has mode.
				if strings.Contains(unsetModes, char) || userHasMode {
					unsetModes = strings.Replace(unsetModes, char, "", -1)
					continue
				}
				setModes = setModes + char
			}
		}
	}

	if len(setModes) >= 1 || len(unsetModes) >= 1 { // If any mode changes
		modeChanges := ""
		if len(unsetModes) > 0 {
			// Unset every mode that needs to be unset.
			for _, mode := range unsetModes {
				msg.User.Modes = strings.Replace(msg.User.Modes, string(mode), "", -1)
			}
			modeChanges = modeChanges + "-" + unsetModes
		}
		if len(setModes) >= 1 {
			// Add modes to user's mode list
			msg.User.Modes = msg.User.Modes + setModes
			modeChanges = modeChanges + "+" + setModes
		}
		// Log changes, and notify client.
		if len(modeChanges) > 0 {
			fmt.Println("Changed modes for", msg.User.Nick, ":: "+modeChanges)
			defer msg.User.Command("MODE", modeChanges)
		}
	}

	if !unknownReached {
		return "", ""
	}

	// We don't use return for this numeric because we'd prefer to send
	// this numeric before notifying client of mode change.
	msg.User.sendNumeric(ERR_USERSDONTMATCH, ":Unknown MODE flag")
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
