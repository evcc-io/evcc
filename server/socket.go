package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/andig/evcc/core"
	"github.com/gorilla/websocket"
)

const (
	// Time allowed to write a message to the peer
	socketWriteTimeout = 10 * time.Second
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

// SocketClient is a middleman between the websocket connection and the hub.
type SocketClient struct {
	hub *SocketHub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte
}

// writePump pumps messages from the hub to the websocket connection.
func (c *SocketClient) writePump() {
	defer c.conn.Close()

	for msg := range c.send {
		if err := c.conn.SetWriteDeadline(time.Now().Add(socketWriteTimeout)); err != nil {
			return
		}
		if err := c.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
			return
		}
	}
}

// ServeWebsocket handles websocket requests from the peer.
func ServeWebsocket(hub *SocketHub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.ERROR.Println(err)
		return
	}
	client := &SocketClient{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	// run writing to client in goroutine
	go client.writePump()
}

// SocketHub maintains the set of active clients and broadcasts messages to the
// clients.
type SocketHub struct {
	// Registered clients.
	clients map[*SocketClient]bool

	// Register requests from the clients.
	register chan *SocketClient

	// Unregister requests from clients.
	unregister chan *SocketClient
}

// NewSocketHub creates a web socket hub that distributes meter status and
// query results for the ui or other clients
func NewSocketHub() *SocketHub {
	return &SocketHub{
		register:   make(chan *SocketClient),
		unregister: make(chan *SocketClient),
		clients:    make(map[*SocketClient]bool),
	}
}

func (h *SocketHub) encode(v core.Param) ([]byte, error) {
	var s string
	switch val := v.Val.(type) {
	case time.Duration:
		// must be before stringer to convert to seconds instead of string
		s = fmt.Sprintf("{\"%s\": %d}", v.Key, int64(val.Seconds()))
	case fmt.Stringer, string:
		s = fmt.Sprintf("{\"%s\": \"%s\"}", v.Key, val)
	case float64:
		s = fmt.Sprintf("{\"%s\": %.3f}", v.Key, val)
	default:
		s = fmt.Sprintf("{\"%s\": %v}", v.Key, val)
	}
	return []byte(s), nil
}

func (h *SocketHub) broadcast(i core.Param) {
	if len(h.clients) > 0 {
		message, err := h.encode(i)
		if err != nil {
			log.FATAL.Fatal(err)
		}

		for client := range h.clients {
			select {
			case client.send <- message:
			default:
				close(client.send)
				delete(h.clients, client)
			}
		}
	}
}

// Run starts data and status distribution
func (h *SocketHub) Run(in <-chan core.Param, triggerChan chan<- struct{}) {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
			triggerChan <- struct{}{} // trigger loadpoint update
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				close(client.send)
				delete(h.clients, client)
			}
		case msg, ok := <-in:
			if !ok {
				return // break if channel closed
			}
			h.broadcast(msg)
		}
	}
}
