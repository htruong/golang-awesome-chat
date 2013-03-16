// Golang HTML5 Server Side Events Example
//
// Run this code like:
//  > go run server.go
//
// Then open up your browser to http://localhost:8000
// Your browser must support HTML5 SSE, of course.

package main

import (
	"./uuid"
	"code.google.com/p/monnand-goconf"
	"encoding/json"
	"flag"
	"fmt"
	//"html/template"
	"log"
	"net/http"
	//"net/url"
	"crypto/md5"
	"io"
	"strings"
	"time"
)

type EventMessage struct {
	Service   bool
	Timestamp int64
	Code      string
	Origin    string
	dest      string
	Payload   string
}

type Session struct {
	id     string
	Nick   string
	Muted  bool
	Admin  bool
	Joined int64
}

type UserListMessage struct {
	Sessions []Session
}

// A single Broker will be created in this program. It is responsible
// for keeping a list of which clients (browsers) are currently attached
// and broadcasting events (messages) to those clients.
//
type Broker struct {

	// Create a map of clients, the keys of the map are the channels
	// over which we can push messages to attached clients.
	clients map[chan string]*Session

	// Config file
	cfg *conf.ConfigFile

	// Banned users
	bannedUsers *map[string]bool

	// Admin users
	adminUsers *map[string]bool

	// Channel into which messages are pushed to be broadcast out
	// to attahed clients.
	messages chan EventMessage

	// Cached messages
	cachedmessages []string
}

// This Broker method starts a new goroutine.  It handles
// the addition & removal of clients, as well as the broadcasting
// of messages out to clients that are currently attached.
//
func (b *Broker) Start() {
	go func() {

		// Loop endlessly
		//
		for {
			log.Printf("Waiting for the next global message...\n")
			//case msg := <-b.messages:
			msg := <-b.messages
			msgToSend, err := json.Marshal((msg))

			log.Printf("Got: %s. Delivering...\n", msgToSend)
			// There is a new message to send.  For each
			// attached client, push the new message
			// into the client's message channel.
			if err == nil {
				sMsgToSend := string(msgToSend)

				if len(msg.dest) == 0 && msg.Code == "msg" {
					if len(b.cachedmessages) > 99 {
						b.cachedmessages = b.cachedmessages[1 : len(b.cachedmessages)-1]
					}
					b.cachedmessages = append(b.cachedmessages, sMsgToSend)
				}

				for s, session := range b.clients {
					if (len(msg.dest) == 0) || (session.id == msg.dest) {
						s <- sMsgToSend
					}
				}
				//log.Printf("Broadcast message to %d clients", len(b.clients))
			}
			//}
		}
	}()
}

// This Broker method handles and HTTP request at the "/events/" URL.
//
func (b *Broker) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	// Make sure that the writer supports flushing.
	//
	f, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}

	cb := w.(http.CloseNotifier).CloseNotify()

	// Create a new channel, over which the broker can
	// send this client messages.
	messageChan := make(chan string)

	// Add this client to the map of those that should
	// receive updates
	newUUID, _ := uuid.GenUUID()
	b.clients[messageChan] = &Session{
		id:     newUUID,
		Muted:  true,
		Admin:  false,
		Nick:   "",
		Joined: time.Now().Unix()}
	log.Printf("Added new client %s", newUUID)

	// Send UUID back to client
	go func() {
		msg, _ := json.Marshal(EventMessage{
			Service:   true,
			Timestamp: time.Now().Unix(),
			Code:      "uuid-return",
			Origin:    "auth",
			dest:      newUUID,
			Payload:   newUUID})
		messageChan <- string(msg)
		log.Printf("Sent new client its uuid %s", newUUID)
		log.Printf(">>>> %d", len(b.cachedmessages))
		for _, val := range b.cachedmessages {
			messageChan <- val
		}
	}()

	// Remove this client from the map of attached clients
	// when `EventHandler` exits.
	go func() {
		<-cb
		nick := b.clients[messageChan].Nick
		delete(b.clients, messageChan)
		log.Printf("Removed client %s", newUUID)
		if len(nick) > 0 {
			b.messages <- EventMessage{
				Service:   true,
				Timestamp: time.Now().Unix(),
				Code:      "part",
				Origin:    "sys",
				dest:      "",
				Payload:   nick}
		}
	}()

	log.Printf("Caught client callback %s", newUUID)

	// Set the headers related to event streaming.
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	// Don't close the connection,
	// sending messages and flushing the response each time
	// there is a new message to send along.
	done := false
	for !done {
		log.Printf("Waiting for the next channel message...\n")
		msg := <-messageChan
		log.Printf("Got: %s. Sending...\n", msg)
		_, err := fmt.Fprintf(w, "data: %s\n\n", msg)
		if err != nil {
			done = true
		} else {
			f.Flush()
		}
	}
}

func AuthHandler(w http.ResponseWriter, r *http.Request, b *Broker) {
	v := r.URL.Query()
	nick := v.Get("nick")
	auth := v.Get("auth")
	uuid := v.Get("uuid")
	secret, err := b.cfg.GetString("server", "secret")
	if err != nil {
		secret = ""
	}
	dryrun, err := b.cfg.GetBool("server", "dryrun")
	if err != nil {
		dryrun = false
	}
	h := md5.New()
	io.WriteString(h, nick)
	io.WriteString(h, secret)
	success := 0
	if auth == string(h.Sum(nil)) || dryrun {
		success = 1

		// announce join
		b.messages <- EventMessage{
			Service:   true,
			Timestamp: time.Now().Unix(),
			Code:      "join",
			Origin:    "sys",
			dest:      "",
			Payload:   nick}
	}

	session := findSession(b, uuid)
	session.Nick = nick
	isBanned := (*b.bannedUsers)[nick]
	isAdmin := (*b.adminUsers)[nick]
	if !isBanned {
		session.Muted = false
	}

	if isAdmin || dryrun {
		session.Admin = true
	}

	msg := EventMessage{
		Service:   false,
		Timestamp: time.Now().Unix(),
		Code:      "msg-receipt",
		Origin:    "sys",
		dest:      uuid,
		Payload:   string(success)}

	msgToSend, _ := json.Marshal((msg))

	w.Header().Set("Content-Type", "text/javascript")
	w.Header().Set("Cache-Control", "no-cache")
	w.Write(msgToSend)
	//fmt.Fprintf(w, "%s", msgToSend)
}

func findSession(b *Broker, uuid string) *Session {
	for _, session := range b.clients {
		if session.id == uuid {
			return session
		}
	}
	return nil
}

func findSessionByNickname(b *Broker, nick string) *Session {
	for _, session := range b.clients {
		if session.Nick == nick {
			return session
		}
	}
	return nil
}

func SendHandler(w http.ResponseWriter, r *http.Request, b *Broker) {
	v := r.URL.Query()
	payload := v.Get("payload")
	uuid := v.Get("uuid")

	ret := "false"
	session := findSession(b, uuid)

	if session != nil && !session.Muted {
		b.messages <- EventMessage{
			Service:   false,
			Timestamp: time.Now().Unix(),
			Code:      "msg",
			Origin:    session.Nick,
			dest:      "",
			Payload:   payload}
		ret = "true"
	}

	msg := EventMessage{
		Service:   true,
		Timestamp: time.Now().Unix(),
		Code:      "msg-receipt",
		Origin:    "sys",
		dest:      uuid,
		Payload:   ret}

	w.Header().Set("Content-Type", "text/javascript")
	w.Header().Set("Cache-Control", "no-cache")
	msgToSend, _ := json.Marshal((msg))

	w.Write(msgToSend)
	//fmt.Fprintf(w, "%s", string(msgToSend))
}

func CommandHandler(w http.ResponseWriter, r *http.Request, b *Broker) {
	v := r.URL.Query()
	payload := v.Get("payload")
	uuid := v.Get("uuid")

	ret := "false"

	session := findSession(b, uuid)

	if !session.Admin {
		return
	}

	// Okay let's do it
	cmd := strings.SplitN(payload, " ", 2)
	ret = "true"
	switch cmd[0] {
	case "ban":
		(*b.bannedUsers)[cmd[1]] = true
		sess := findSessionByNickname(b, cmd[1])
		sess.Muted = true
	case "unban":
		delete(*b.bannedUsers, cmd[1])
		sess := findSessionByNickname(b, cmd[1])
		sess.Muted = false
	case "op":
		(*b.adminUsers)[cmd[1]] = true
		sess := findSessionByNickname(b, cmd[1])
		sess.Admin = true
	case "unop":
		delete(*b.adminUsers, cmd[1])
		sess := findSessionByNickname(b, cmd[1])
		sess.Admin = false
	default:
		ret = "false"
	}

	defer func() {
		msg := EventMessage{
			Service:   true,
			Timestamp: time.Now().Unix(),
			Code:      "cmd-receipt",
			Origin:    "sys",
			dest:      uuid,
			Payload:   ret}

		w.Header().Set("Content-Type", "text/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		msgToSend, _ := json.Marshal((msg))

		w.Write(msgToSend)
	}()

}

func UsersListHandler(w http.ResponseWriter, r *http.Request, b *Broker) {

	sessions := make([]Session, 0, 100)
	for _, session := range b.clients {
		if len(session.Nick) > 0 {
			sessions = append(sessions, *session)
		}
	}

	msg := UserListMessage{
		Sessions: sessions,
	}

	w.Header().Set("Content-Type", "text/javascript")
	w.Header().Set("Cache-Control", "no-cache")
	msgToSend, _ := json.Marshal((msg))

	w.Write(msgToSend)
}

// Main routine
//
func main() {
	var configFile *string = flag.String("f", "lily.conf", "Config file")
	var port *string = flag.String("p", "8080", "Server Port")
	var verbose *bool = flag.Bool("v", false, "Verbose")

	flag.Parse()

	if *verbose {
		// Do some studpid shit
	}

	cfg, _ := conf.ReadConfigFile(*configFile)
	admins, _ := cfg.GetString("room", "admins")

	adminUsers := make(map[string]bool)
	rootAdmins := strings.Split(admins, ";")
	for _, i := range rootAdmins {
		adminUsers[i] = true
	}

	bannedUsers := make(map[string]bool)

	cachedmessages := make([]string, 0, 100)

	// Make a new Broker instance
	b := &Broker{
		clients:        make(map[chan string]*Session),
		cfg:            cfg,
		bannedUsers:    &bannedUsers,
		adminUsers:     &adminUsers,
		messages:       make(chan EventMessage),
		cachedmessages: cachedmessages,
	}

	// Start processing events
	b.Start()

	// Make b the HTTP handler for "/events/".  It can do
	// this because it has a ServeHTTP method.  That method
	// is called in a separate goroutine for each
	// request to "/events/".
	http.Handle("/chat/events/", b)

	http.Handle(
		"/chat/auth/",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			AuthHandler(w, r, b)
		}))

	http.Handle(
		"/chat/send/",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			SendHandler(w, r, b)
		}))

	http.Handle(
		"/chat/command/",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			CommandHandler(w, r, b)
		}))

	http.Handle(
		"/chat/userslist/",
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			UsersListHandler(w, r, b)
		}))

	// When we get a request at "/", call `MainPageHandler`
	// in a new goroutine.
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))

	// Start the server and listen forever on port 8000.
	http.ListenAndServe(":"+*port, nil)
}
