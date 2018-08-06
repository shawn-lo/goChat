package main

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"fmt"
	"math/rand"
	"strconv"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Client is a middleman between the websocket connection and the hub.
type Agent struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	recv chan []byte

	ip string

	sid string
}

func (a *Agent) HashKey() string {
	return a.ip + a.sid
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (a *Agent) readPump() {
	defer func() {
		a.hub.aUnregister <- a
		a.conn.Close()
	}()
	a.conn.SetReadLimit(maxMessageSize)
	// This used to set read deadline to a time less than next expected pong.
	a.conn.SetReadDeadline(time.Now().Add(pongWait))
	// This will be called below, in a.conn.ReadMessage(). It used to refresh pong time when received pong successfully.
	a.conn.SetPongHandler(func(string) error { a.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		fmt.Println("test")
		_, message, err := a.conn.ReadMessage()
		c := a.hub.agentMapToCus[a]
		if err != nil {
			fmt.Println("Pong missed. ")
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		if c != nil {
			fmt.Println(c.HashKey())
			c.recv<- message
		} else {
			fmt.Println("No customer online. Send message failed.")
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (a *Agent) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		a.conn.Close()
	}()
	for {
		select {
		case message, ok := <-a.recv:
			a.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				a.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := a.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(a.recv)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-a.recv)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			a.conn.SetWriteDeadline(time.Now().Add(writeWait))
			// Send regular ping
			if err := a.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Println("Agent. From hub to ws, disconnect")
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWsa(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	agent := &Agent{hub: hub, conn: conn, recv: make(chan []byte, 256), ip: "testIP", sid: strconv.Itoa(rand.Int())}
	agent.hub.aRegister <- agent

	go agent.writePump()
	go agent.readPump()
}