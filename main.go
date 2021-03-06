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
	Connection   net.Listener
}

func (server *Server) nickExists(nick string) (exists bool, registered bool, user ircUser) {
	// Check if wanted nickname is in use. Case-insensitive
	// Preferred usage: User-input when they may enter case insensitive nicks.
	// BENCHMARK this, see if equalfold is resource intensive. -- Syed
	if u, ok := server.Clients[nick]; ok { // Try the easy way first.
		exists, registered, user = true, true, *u
		return
	}

	for _, v := range server.Unregistered {
		if strings.EqualFold(v.Nick, nick) {
			exists, user = true, *v
			return
		}
	}
	for k, v := range server.Clients {
		if strings.EqualFold(k, nick) {
			exists, registered, user = true, true, *v
			return
		}
	}
	return
}

type ircMessage struct {
	User    *ircUser
	Command string
	Payload []string
	Server  *Server
}

func main() {
	// Check if root and if it is, send a warning.
	if syscall.Geteuid() == 0 {
		fmt.Println("WARNING: You're running as root, please don't do this if you can run as another user.")
	}

	// Initialize a new server
	server := Server{}
	server.Name = "Syed's FunHouse"
	server.Host = "InitialIRCD.testserver.net"
	server.Unregistered = make(map[*net.Conn]*ircUser)
	server.Clients = make(map[string]*ircUser)

	// Start listening on port 6667. More ports in the future.
	conn, err := net.Listen("tcp", ":6667")
	if err != nil {
		log.Fatal(err)
	}
	server.Connection = conn

	// Create a channel that handles messages for the server.
	msgchan := make(chan ircMessage)
	go handleMessages(msgchan)

	for {
		conn, err := server.Connection.Accept()
		if err != nil {
			fmt.Printf("%v\n - Error is in main()", err)
			break
		}
		fmt.Printf("%s: %v <-> %v\n", "New connection accepted", conn.LocalAddr(), conn.RemoteAddr())
		// On connect, send connection info, message channel, and server
		go handleConnection(conn, msgchan, &server)
	}
}

func handleConnection(c net.Conn, msgchan chan<- ircMessage, server *Server) {
	raw := bufio.NewReader(c)
	b := bufio.NewReader(io.LimitReader(raw, 1024))

	// Initialize User
	user := ircUser{}
	user.Nick = "AUTH"
	user.Conn = c
	user.Server = server

	user.Writer = make(chan string)
	killswitch := make(chan bool)

	// Start creating the message.
	var message ircMessage
	message.User, message.Server = &user, server

	go func() {
		for {
			select {
			case <-killswitch:
				fmt.Println("Killing writer goroutine for " + message.User.Nick)
				return
			default:
				write := <-user.Writer
				bytes := []byte(write + "\r\n")
				_, err := user.Conn.Write(bytes)
				if err != nil {
					fmt.Printf("Write err: %v\n", err)
					user.Conn.Close()
					return
				}
			}
		}
	}()

	for {
		line, _, err := b.ReadLine()
		if err != nil { // EOF, or worse
			fmt.Printf("%v\n", err)
			killswitch <- true
			c.Close()
			return
		}
		// Split the incoming message into command and payload and send to message channel
		lnsplit := strings.Split(string(line), " ")
		message.Command = strings.ToUpper(lnsplit[0]) // Commands are stored in uppercase
		if len(lnsplit) > 1 {
			lnsplit[1] = strings.TrimPrefix(lnsplit[1], ":") // Remove ":" prefix
			message.Payload = lnsplit[1:]
		}
		msgchan <- message
		// Remove payload and command from memory.
		message.Payload, message.Command = []string{}, ""
	}
	// log.Printf("Connection from %v closed.", c.RemoteAddr())
}

func handleMessages(msgchan <-chan ircMessage) {
	for msg := range msgchan {
		// Rate limiter
		// if msg.User.reachedLimit() {
		// 	msg.User.raw(":"+msg.User.Host, "QUIT", ":Excess flood")
		// 	msg.User.Conn.Close()
		// 	msg.User.deleteUser()
		// }
		fmt.Printf("%s :: %s || %s\n", msg.User.Nick, msg.Command, msg.Payload)
		msg.handleCommand()
	}
}

type CommandInfo struct {
	// Will put more here in future.
	run     func(*ircMessage) (string, string) // Holds a pointer to function call
	minimum int                                // Minimum parameters allowed
}

func (msg *ircMessage) handleCommand() {
	// Call related function
	// List of all handlers based on the scommand sent by clients.
	commands := map[string]CommandInfo{
		// Command : Function, minimum parameters.
		"USER":     CommandInfo{IRC_USER, 4},
		"NICK":     CommandInfo{IRC_NICK, 1},
		"CAP":      CommandInfo{IRC_CAP, 0},
		"QUIT":     CommandInfo{IRC_QUIT, 0},
		"PING":     CommandInfo{IRC_PONG, 0},
		"MODE":     CommandInfo{IRC_MODE, 1},
		"USERHOST": CommandInfo{IRC_USERHOST, 1},
		"ISON":     CommandInfo{IRC_ISON, 1},
		"TIME":     CommandInfo{IRC_TIME, 0},
	}
	if ircCommand, found := commands[msg.Command]; !found {
		msg.User.sendNumeric(ERR_UNKNOWNCOMMAND, msg.Command+" :This command is unknown or unsupported.")
		return
	} else {
		if len(msg.Payload) >= ircCommand.minimum {
			retCode, retMsg := ircCommand.run(msg)
			if retCode != "" && retMsg != "" {
				msg.User.sendNumeric(retCode, retMsg)
			}
		} else {
			msg.User.sendNumeric(ERR_NEEDMOREPARAMS, msg.Command+" :Not enough parameters")
		}
	}
}
