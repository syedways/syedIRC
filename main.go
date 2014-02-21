package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"net"
	"strings"
	"syscall"
)

type Server struct {
	Name         string
	Host         string
	Unregistered map[*net.Conn]*ircUser
	Clients      map[string]*ircUser
}

type ircMessage struct {
	User    *ircUser
	Command string
	Payload []string
	Server  Server
}

func main() {
	// Check if root and if it is, send a warning.
	if syscall.Geteuid() == 0 {
		fmt.Println("WARNING: You're running as root, please don't do this if you can run as another user.")
	}

	// Start listening on port 6667. More ports in the future.
	ln, err := net.Listen("tcp", ":6667")
	if err != nil {
		log.Fatal(err)
	}

	// Create a channel that handles messages.
	msgchan := make(chan ircMessage)
	go handleMessages(msgchan)

	// Initialize a new server
	server := Server{}
	server.Name = "Syed's FunHouse"
	server.Host = "InitialIRCD.testserver.net"
	server.Unregistered = make(map[*net.Conn]*ircUser)
	server.Clients = make(map[string]*ircUser)

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Printf("%v\n - Error is in main()", err)
			continue
		}
		fmt.Printf("%s: %v <-> %v\n", "New connection accepted", conn.LocalAddr(), conn.RemoteAddr())
		// On connect, send connection info, message channel, and server
		go handleConnection(conn, msgchan, server)
	}
}

func handleConnection(c net.Conn, msgchan chan<- ircMessage, server Server) {
	raw := bufio.NewReader(c)
	b := bufio.NewReader(io.LimitReader(raw, 1024))

	// Initialize User
	user := ircUser{}
	user.Nick = "AUTH"
	user.Writer = make(chan string)
	user.Conn = c
	user.Server = server
	go user.handleWrite(user.Writer)

	// Start creating the message.
	var message ircMessage
	message.User, message.Server = &user, server

	for {
		line, _, err := b.ReadLine()
		if err != nil { // EOF, or worse
			fmt.Printf("%v\n", err)
			c.Close()
			break
		}
		// Split the incoming message into command and payload and send to message channel
		lnsplit := strings.Split(string(line), " ")
		message.Command = lnsplit[0]
		if len(lnsplit) > 1 {
			lnsplit[1] = strings.TrimPrefix(lnsplit[1], ":") // Remove ":" prefix
			message.Payload = lnsplit[1:]
		}
		msgchan <- message
	}
	// log.Printf("Connection from %v closed.", c.RemoteAddr())
}

func (user *ircUser) handleWrite(writer <-chan string) {
	for write := range writer {
		fmt.Println(write)
		bytes := []byte(write + "\r\n")
		_, err := user.Conn.Write(bytes)
		if err != nil {
			fmt.Printf("Write err: %v\n", err)
			user.Conn.Close()
			return
		}
	}
}

func handleMessages(msgchan <-chan ircMessage) {
	for msg := range msgchan {
		// Get updated user
		msg.User = msg.User.getUser()
		// // Rate limiter
		// if msg.User.reachedLimit() {
		// 	msg.User.raw(":"+msg.User.Host, "QUIT", ":Excess flood")
		// 	msg.User.Conn.Close()
		// 	msg.User.deleteUser()
		// }
		fmt.Printf("%s :: %s || %s\n", msg.User.Nick, msg.Command, msg.Payload)
		// List of all handlers based on the scommand sent by clients.
		commands := map[string]interface{}{
			"USER":     IRC_USER,
			"NICK":     IRC_NICK,
			"CAP":      IRC_CAP,
			"QUIT":     IRC_QUIT,
			"PING":     IRC_PONG,
			"MODE":     IRC_MODE,
			"USERHOST": IRC_USERHOST,
			"ISON":     IRC_ISON,
		}
		if f, found := commands[msg.Command]; found {
			msg.handleCommand(f)
		} else {
			msg.User.sendNumeric(ERR_UNKNOWNCOMMAND, msg.Command+" :This command is unknown or unsupported.")
		}
	}
}

func (msg *ircMessage) handleCommand(f interface{}) {
	// Call related function
	f.(func(*ircMessage))(msg)
}
