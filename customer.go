package main

import (
	"bytes"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"fmt"
)

// Client is a middleman between the websocket connection and the hub.
type Customer struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	recv chan []byte

	ip string

	lastRecvTime time.Time
}

func (c *Customer) HashKey() string {
	return c.ip
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Customer) readPump() {
	defer func() {
		c.hub.cUnregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		diff := time.Since(c.lastRecvTime)
		if diff < VALID_DURATION {
			// in valid duration
			c.lastRecvTime = time.Now()
		} else {
			break
		}
		fmt.Print("The diff is ", time.Since(c.lastRecvTime))

		a := c.hub.cusMapToAgent[c]
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		a.recv <- message
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Customer) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.recv:
			fmt.Print(message)
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message.
			n := len(c.recv)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.recv)
			}

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				fmt.Println("Customer. From hub to ws, disconnect")
				return
			}
		}
	}
}

// serveWs handles websocket requests from the peer.
func serveWsc(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	customer := &Customer{hub: hub, conn: conn, recv: make(chan []byte, 256), ip: "testIPCustomer", lastRecvTime: time.Now()}
	customer.hub.cRegister <- customer

	go customer.writePump()
	go customer.readPump()
}