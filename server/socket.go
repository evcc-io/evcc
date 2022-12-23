package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/util"
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
	defer func() {
		c.conn.Close()
		c.hub.unregister <- c
	}()

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

func encode(v interface{}) (string, error) {
	var s string
	switch val := v.(type) {
	case time.Time:
		var b []byte
		if !val.IsZero() {
			b, _ = val.MarshalText()
		}
		s = fmt.Sprintf(`"%s"`, string(b))
	case time.Duration:
		// must be before stringer to convert to seconds instead of string
		s = fmt.Sprintf("%d", int64(val.Seconds()))
	case float64:
		if math.IsNaN(val) {
			s = "null"
		} else {
			s = fmt.Sprintf("%.5g", val)
		}
	default:
		if b, err := json.Marshal(v); err == nil {
			s = string(b)
		}
	}
	return s, nil
}

func kv(p util.Param) string {
	val, err := encode(p.Val)
	if err != nil {
		panic(err)
	}

	var msg strings.Builder
	msg.WriteString("\"")
	if p.Loadpoint != nil {
		msg.WriteString(fmt.Sprintf("loadpoints.%d.", *p.Loadpoint))
	}
	msg.WriteString(p.Key)
	msg.WriteString("\":")
	msg.WriteString(val)

	return msg.String()
}

func (h *SocketHub) welcome(client *SocketClient, params []util.Param) {
	h.clients[client] = true

	var msg strings.Builder
	msg.WriteString("{")
	for _, p := range params {
		if msg.Len() > 1 {
			msg.WriteString(",")
		}
		msg.WriteString(kv(p))
	}
	msg.WriteString("}")

	select {
	case client.send <- []byte(msg.String()):
	default:
		close(client.send)
	}
}

func (h *SocketHub) broadcast(p util.Param) {
	if len(h.clients) > 0 {
		msg := "{" + kv(p) + "}"

		for client := range h.clients {
			select {
			case client.send <- []byte(msg):
			default:
				h.unregister <- client
			}
		}
	}
}

// Run starts data and status distribution
func (h *SocketHub) Run(in <-chan util.Param, cache *util.Cache) {
	for {
		select {
		case client := <-h.register:
			h.welcome(client, cache.All())
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
