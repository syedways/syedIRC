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
	user.Server = server
	return user
}

func mockWriter(writer <-chan string) {
	for write := range writer {
		write = write + "S" // Do nothing with it.
	}
}

func mock_message(line string) (msg ircMessage) {
	lnsplit := strings.Split(string(line), " ")
	lnsplit[1] = strings.TrimPrefix(lnsplit[1], ":") // Remove ":" prefix
	msg.Command, msg.Payload = lnsplit[0], lnsplit[1:]
	return
}

func Test_Registration(t *testing.T) {
	testuser := mock_user()
	nickmsg := mock_message("NICK Test")
	usermsg := mock_message("USER Volt 0 * :...")
	nickmsg.User, usermsg.User = &testuser, &testuser

	nickmsg.handleCommand(IRC_NICK)
	nickmsg.User = nickmsg.User.getUser()
	if nickmsg.User.Nick == "Test" {
		t.Log("NICK Test Passed")
	}
	usermsg.handleCommand(IRC_USER)
	usermsg.User = usermsg.User.getUser()
	if usermsg.User.User == "Volt" {
		t.Log("USER Test Passed")
	}
}
