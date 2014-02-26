package main

import (
	"net"
	"strings"
	"testing"
)

func mock_user() (user ircUser) {
	server := Server{}
	server.Name = "TestIRCd"
	server.Host = "TestIRCd.testserver.net"
	server.Unregistered = make(map[*net.Conn]*ircUser)
	server.Clients = make(map[string]*ircUser)

	user.Nick = "AUTH"
	user.Writer = make(chan string)
	go mockWriter(user.Writer)
	user.Conn, _ = net.Dial("tcp", "localhost:80") // Fake a net.Conn
	user.Server = &server
	return user
}

func mockWriter(writer <-chan string) {
	for write := range writer {
		write = write + "S" // Do nothing with it.
	}
}

func mock_message(line string, user *ircUser) (msg ircMessage) {
	// Initialize user.
	if user.Nick == "" {
		testuser := mock_user()
		msg.User, msg.Server = &testuser, testuser.Server
	} else {
		msg.User, msg.Server = user, user.Server
	}

	lnsplit := strings.Split(string(line), " ")
	lnsplit[1] = strings.TrimPrefix(lnsplit[1], ":") // Remove ":" prefix
	msg.Command, msg.Payload = lnsplit[0], lnsplit[1:]
	return
}

func Test_Registration(t *testing.T) {
	nickmsg := mock_message("NICK Test", &ircUser{})
	nickmsg.handleCommand()
	if nickmsg.User.Nick == "Test" {
		t.Log("NICK Test has passed")
	} else {
		t.Error("NICK Test has failed.")
	}

	usermsg := mock_message("USER TestUser 0 * :...", nickmsg.User)
	usermsg.handleCommand()
	if usermsg.User.User == "TestUser" && usermsg.User.Realname == "..." &&
		usermsg.User.Nick == "Test" && usermsg.User.Host == "Test!~TestUser@127.0.0.1" {
		t.Log("USER Test has Passed")
	} else {
		t.Error("USER Test has failed.")
	}
}

func Test_Nick_Change(t *testing.T) {
	// Register User
	regmsg := mock_message("NICK Test", &ircUser{})
	regmsg.handleCommand()
	reg2msg := mock_message("USER TestUser 0 * :...", regmsg.User)
	reg2msg.handleCommand()

	// Change registered user's nick
	nickmsg := mock_message("NICK Test2", reg2msg.User)
	nickmsg.handleCommand()
	if _, ok := nickmsg.Server.Clients[nickmsg.User.Nick]; ok && nickmsg.User.Nick == "Test2" {
		t.Log("NICK Change Test has passed.")
	} else {
		t.Errorf("NICK Change Test has failed.")
	}
}

func Test_Nick_Regex(t *testing.T) {
	// All of the nicknames below should lead to an erroneous nickname response.
	erroneousnicks := []string{
		"^123#5678",  // Has disallowed character (#)
		"&#23",       // Starts with disallowed character (&)
		"SliCk%$",    // Has disallowed characters (%$)
		"R123333333", // Is too long, 10 characters (9 character is max)
		"1R23",       // Starts with disallowed character (digit)
	}
	success := 0
	for nick := range erroneousnicks {
		regmsg := mock_message("NICK "+erroneousnicks[nick], &ircUser{})
		regmsg.User.Writer = make(chan string)
		go func() {
			for {
				write := <-regmsg.User.Writer
				if strings.Split(write, " ")[3] == erroneousnicks[nick] {
					success++
				}
			}
		}()
		regmsg.handleCommand()
	}
	if success == len(erroneousnicks) {
		t.Log("NICK Regex Test has passed.")
	} else {
		t.Error("NICK Regex Test has failed.")
	}
}
