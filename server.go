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
	"strconv"
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
	UID    string
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

	votes  chan bool
	voted  *[]string
	voting *bool
	dryrun *bool

	banQueue   chan string
	saveConfig chan bool
	configFile string

	// Cached messages
	cachedmessages []EventMessage
}

func findInSlice(slice []string, value string) int {
	for p, v := range slice {
		if v == value {
			return p
		}
	}
	return -1
}

// This Broker method starts a new goroutine.  It handles
// the addition & removal of clients, as well as the broadcasting
// of messages out to clients that are currently attached.
//
func (b *Broker) Start() {
	go func() {
		for {
			<-b.saveConfig
			b.cfg.AddSection("runtime")
			allBanned := make([]string, 0, 100)
			for k, _ := range *b.bannedUsers {
				allBanned = append(allBanned, k)
			}
			b.cfg.AddOption("runtime", "banned",
				strings.Join(allBanned, ";"))

			allAdmins := make([]string, 0, 100)
			for k, _ := range *b.adminUsers {
				allAdmins = append(allAdmins, k)
			}
			b.cfg.AddOption("runtime", "admins",
				strings.Join(allAdmins, ";"))

			b.cfg.WriteConfigFile(b.configFile, 0,
				fmt.Sprintf("File autogenerated on %d", time.Now().Unix()))
		}
	}()

	go func() {
		for {
			booter := <-b.banQueue
			log.Printf("Got ban request for %s\n", booter)
			kicked := false
			for _, session := range b.clients {
				if session.Nick == booter {
					session.Muted = true
				}
			}
			log.Printf("Requested all sessions of %s to quit\n", booter)

			if kicked {
				b.messages <- EventMessage{
					Service:   true,
					Timestamp: time.Now().Unix(),
					Code:      "msg",
					Origin:    "sys",
					dest:      "",
					Payload:   fmt.Sprintf("%s is now muted!", booter)}
			}
			log.Printf("Announced %s is gone\n", booter)

			// Clean up
			for k, val := range b.cachedmessages {
				if val.Origin == booter {
					copy(
						b.cachedmessages[k:],
						b.cachedmessages[k+1:])
					b.cachedmessages = b.cachedmessages[:len(b.cachedmessages)-1]
				}
			}

			b.saveConfig <- true
		}
	}()

	go func() {

		// Loop endlessly
		//
		for {
			log.Printf("Waiting for the next global message...\n")
			//case msg := <-b.messages:
			msg := <-b.messages

			if len(b.cachedmessages) > 3 {
				if (b.cachedmessages[len(b.cachedmessages)-1].Origin == msg.Origin) &&
					(b.cachedmessages[len(b.cachedmessages)-2].Origin == msg.Origin) &&
					(b.cachedmessages[len(b.cachedmessages)-3].Origin == msg.Origin) {
					// Spam
					// b.banQueue <- msg.Origin
					continue
				}
			}

			msgToSend, err := json.Marshal((msg))

			log.Printf("Got: %s. Delivering...\n", msgToSend)
			fmt.Printf("%s\n", msgToSend)
			// There is a new message to send.  For each
			// attached client, push the new message
			// into the client's message channel.
			if err == nil {

				sMsgToSend := string(msgToSend)

				if len(msg.dest) == 0 && msg.Code == "msg" {
					if len(b.cachedmessages) > 99 {
						b.cachedmessages = b.cachedmessages[1 : len(b.cachedmessages)-1]
					}
					b.cachedmessages = append(b.cachedmessages, msg)
				}

				for s, session := range b.clients {
					if (len(msg.dest) == 0) || (session.id == msg.dest) {
						//log.Printf("Delivering to %s %s", session.id, session.Nick)
						s <- sMsgToSend
					}
				}
				log.Printf("Broadcast message to %d clients", len(b.clients))
			} else {

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
	newUID, _ := uuid.GenUUID()

	b.clients[messageChan] = &Session{
		id:     newUUID,
		UID:    newUID,
		Muted:  true,
		Admin:  false,
		Nick:   "",
		Joined: time.Now().Unix(),
	}
	//log.Printf("Added new client %s", newUUID)

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
		//log.Printf("Waiting for the next channel message for %s...\n", newUUID)
		msg := <-messageChan
		//log.Printf("Got: %s. Sending...\n", msg)
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

	success := false

	defer func() {
		msg := EventMessage{
			Service:   false,
			Timestamp: time.Now().Unix(),
			Code:      "auth-result",
			Origin:    "sys",
			dest:      uuid,
			Payload:   fmt.Sprintf("%t", success)}

		msgToSend, _ := json.Marshal((msg))

		w.Header().Set("Content-Type", "text/javascript")
		w.Header().Set("Cache-Control", "no-cache")
		w.Write(msgToSend)
	}()

	secret, err := b.cfg.GetString("server", "secret")
	if err != nil {
		secret = ""
	}

	h := md5.New()
	io.WriteString(h, nick)
	io.WriteString(h, secret)

	t := h.Sum(nil)
	expected := fmt.Sprintf("%x", t)
	if auth == expected || *b.dryrun {
		isBanned := (*b.bannedUsers)[nick]
		isAdmin := (*b.adminUsers)[nick]

		_, session := findSession(b, uuid)

		if isAdmin {
			session.Admin = true
		}

		if !isBanned {
			session.Muted = false
		} else if !isAdmin {
			return
		}

		success = true

		// announce join
		b.messages <- EventMessage{
			Service:   true,
			Timestamp: time.Now().Unix(),
			Code:      "join",
			Origin:    "sys",
			dest:      "",
			Payload:   nick}

		session.Nick = nick

	} else {
		log.Printf("Auth failed for %s, Wanted %s, got %s.\n", nick, expected, auth)
	}
	//fmt.Fprintf(w, "%s", msgToSend)
}

func findSession(b *Broker, uuid string) (chan string, *Session) {
	for channel, session := range b.clients {
		if session.id == uuid {
			return channel, session
		}
	}
	return nil, nil
}

func findSessionByNickname(b *Broker, nick string) *Session {
	for _, session := range b.clients {
		if session.Nick == nick {
			return session
		}
	}
	return nil
}

func findSessionByUID(b *Broker, uid string) *Session {
	for _, session := range b.clients {
		if session.UID == uid {
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
	_, session := findSession(b, uuid)

	defer func() {

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

	}()

	if session == nil {
		log.Printf("Session uuid %s not found\n", uuid)
		return
	}

	if !session.Muted {
		b.messages <- EventMessage{
			Service:   false,
			Timestamp: time.Now().Unix(),
			Code:      "msg",
			Origin:    session.Nick,
			dest:      "",
			Payload:   payload}
		ret = "true"
	} else {
		ret = "You are muted"
	}
}

func CommandHandler(w http.ResponseWriter, r *http.Request, b *Broker) {
	v := r.URL.Query()
	payload := v.Get("payload")
	uuid := v.Get("uuid")

	ret := "false"

	defer func() {
		log.Printf("%s '%s' returns %s\n", uuid, payload, ret)
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

	targetchan, session := findSession(b, uuid)

	if session == nil {
		return
	}

	if session.Muted && !session.Admin {
		ret = "You do not have permission to request commands."
		return
	}

	// Okay let's do it
	cmd := strings.SplitN(payload, " ", 2)

	if !session.Admin && !strings.HasPrefix(cmd[0], "_") {
		log.Printf("%s wanted to do an administrative command... Nope!\n", session.Nick)
		return
	}

	ret = "true"
	log.Printf("%s: Executing '%s'!\n", session.Nick, payload)
	switch cmd[0] {
	case "ban":
		target := cmd[1]
		if cmd[1][0] == '#' {
			target = findSessionByUID(b, cmd[1][1:]).Nick
		}
		(*b.bannedUsers)[target] = true
		b.banQueue <- target

	case "unban":
		delete(*b.bannedUsers, cmd[1])
		b.saveConfig <- true

	case "op":
		(*b.adminUsers)[cmd[1]] = true
		sess := findSessionByNickname(b, cmd[1])
		if sess != nil {
			sess.Admin = true
		}
		b.saveConfig <- true

	case "_getoldmessages":
		if num, err := strconv.Atoi(cmd[1]); err == nil {
			if num > len(b.cachedmessages) {
				num = len(b.cachedmessages)
			}
			//log.Printf(">>>> %d", num)

			for _, val := range b.cachedmessages[len(b.cachedmessages)-num:] {
				msg, _ := json.Marshal(val)
				targetchan <- string(msg)
			}
			ret = ""
		}

	case "unop":
		delete(*b.adminUsers, cmd[1])
		sess := findSessionByNickname(b, cmd[1])
		if sess != nil {
			sess.Admin = false
		}
		b.saveConfig <- true

	case "_listbanned":
		allBanned := make([]string, 0, 100)
		for k, _ := range *b.bannedUsers {
			allBanned = append(allBanned, k)
		}
		ret = strings.Join(allBanned, ";")

	case "_ban":
		if *b.voting {
			ret = "Another voting session is in progress."
			return
		}

		if len(findLegitSessions(b.clients)) < 5 && !*b.dryrun {
			log.Printf("%s wants to call vote, but not enough people are in... Nope!\n", session.Nick)
			ret = "Not enough people to call vote"
			return
		}

		go func() {
			target := cmd[1]
			if cmd[1][0] == '#' {
				targetsess := findSessionByUID(b, cmd[1][1:])
				if targetsess != nil {
					target = targetsess.Nick
				} else {
					ret = "Session not found"
					return
				}
			}
			votes := 1
			votes_pass := len(findLegitSessions(b.clients)) * 3 / 4
			b.messages <- EventMessage{
				Service:   true,
				Timestamp: time.Now().Unix(),
				Code:      "msg",
				Origin:    "sys",
				dest:      "",
				Payload: fmt.Sprintf("%s: vote with /_yes or /_no in the next 30 secs to boycott %s. Pass: at least %d yes.",
					session.Nick, target, votes_pass)}
			(*b.voting) = true
			(*b.voted) = make([]string, 100)
			*b.voted = append(*b.voted, session.Nick)
			done := false
			for !done {
				select {
				case <-b.votes:
					votes++
					if votes >= votes_pass {
						done = true
					}
				case <-time.After(30 * time.Second):
					done = true
				}
			}

			if votes >= votes_pass {
				b.messages <- EventMessage{
					Service:   true,
					Timestamp: time.Now().Unix(),
					Code:      "msg",
					Origin:    "sys",
					dest:      "",
					Payload:   fmt.Sprintf("Vote passed! Banning %s...", cmd[1])}

				(*b.bannedUsers)[target] = true
				b.banQueue <- target
			} else {
				b.messages <- EventMessage{
					Service:   true,
					Timestamp: time.Now().Unix(),
					Code:      "msg",
					Origin:    "sys",
					dest:      "",
					Payload:   fmt.Sprintf("Not enough people voted yes. %s stays here", target)}
			}
			(*b.voting) = false

		}()

	case "_yes", "_no":
		ret = "Your answer has been registered."
		ans := false

		if cmd[0] == "_yes" {
			ans = true
		}

		if !*b.voting {
			ret = "No voting session in progress."
			return
		}

		if findInSlice(*b.voted, session.Nick) > -1 {
			ret = "You have already voted"
			return
		} else {
			*b.voted = append(*b.voted, session.Nick)
		}

		b.messages <- EventMessage{
			Service:   true,
			Timestamp: time.Now().Unix(),
			Code:      "msg",
			Origin:    "sys",
			dest:      "",
			Payload:   fmt.Sprintf("One person voted [%t].", ans)}

		if ans == true {
			b.votes <- true
		}

	default:
		ret = "You can do /_ban, /_listbanned."
	}

}

func findLegitSessions(all map[chan string]*Session) []Session {
	sessions := make([]Session, 0, 100)
	seenNicks := make([]string, 0, 100)
	for _, session := range all {
		if len(session.Nick) > 0 {
			if findInSlice(seenNicks, session.Nick) < 0 {
				sessions = append(sessions, *session)
				seenNicks = append(seenNicks, session.Nick)
			}
		}
	}
	return sessions
}

func UsersListHandler(w http.ResponseWriter, r *http.Request, b *Broker) {
	msg := UserListMessage{
		Sessions: findLegitSessions(b.clients),
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
	var wwwroot *string = flag.String("r", "/usr/local/share/chatserver", "Chatserver static root")
	var verbose *bool = flag.Bool("v", false, "Verbose")

	flag.Parse()

	if *verbose {
		// Do some studpid shit
	}

	cfg, _ := conf.ReadConfigFile(*configFile)

	adminUsers := make(map[string]bool)
	admins, _ := cfg.GetString("room", "admins")
	rootAdmins := strings.Split(admins, ";")
	for _, i := range rootAdmins {
		adminUsers[i] = true
	}

	addtionalAdmins, _ := cfg.GetString("runtime", "admins")
	addAdmins := strings.Split(addtionalAdmins, ";")
	for _, i := range addAdmins {
		adminUsers[i] = true
	}

	dryrun, err := cfg.GetBool("server", "dryrun")
	if err != nil {
		dryrun = false
	}

	bannedUsers := make(map[string]bool)
	banned, _ := cfg.GetString("runtime", "banned")
	bannedSplice := strings.Split(banned, ";")
	for _, i := range bannedSplice {
		bannedUsers[i] = true
	}

	voted := make([]string, 100)
	voting := false

	cachedmessages := make([]EventMessage, 0, 100)

	// Make a new Broker instance
	b := &Broker{
		clients:        make(map[chan string]*Session),
		cfg:            cfg,
		configFile:     *configFile,
		bannedUsers:    &bannedUsers,
		adminUsers:     &adminUsers,
		messages:       make(chan EventMessage),
		cachedmessages: cachedmessages,
		dryrun:         &dryrun,
		votes:          make(chan bool),
		voted:          &voted,
		voting:         &voting,
		banQueue:       make(chan string),
		saveConfig:     make(chan bool),
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
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(*wwwroot))))

	// Start the server and listen forever on port 8000.
	http.ListenAndServe(":"+*port, nil)
}
