package server

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/util"
)

const (
	// Time allowed to write a message to the peer
	socketWriteTimeout = 10 * time.Second
)

// socketSubscriber is a middleman between the websocket connection and the hub.
type socketSubscriber struct {
	send      chan []byte
	closeSlow func()
}

func writeTimeout(ctx context.Context, timeout time.Duration, c *websocket.Conn, msg []byte) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return c.Write(ctx, websocket.MessageText, msg)
}

// SocketHub maintains the set of active clients and broadcasts messages to the
// clients.
type SocketHub struct {
	mu          sync.RWMutex
	register    chan *socketSubscriber
	subscribers map[*socketSubscriber]struct{}
}

// NewSocketHub creates a web socket hub that distributes meter status and
// query results for the ui or other clients
func NewSocketHub() *SocketHub {
	return &SocketHub{
		register:    make(chan *socketSubscriber, 1),
		subscribers: make(map[*socketSubscriber]struct{}),
	}
}

// ServeWebsocket handles websocket requests from the peer.
func (h *SocketHub) ServeWebsocket(w http.ResponseWriter, r *http.Request) {
	acceptOptions := &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	}

	// Safari deflate message compression is broken, enable for others
	// see: https://github.com/gorilla/websocket/issues/731
	ua := strings.ToLower(r.Header.Get("User-Agent"))
	if strings.Contains(ua, "chrome") || strings.Contains(ua, "firefox") {
		acceptOptions.CompressionMode = websocket.CompressionContextTakeover
	}

	conn, err := websocket.Accept(w, r, acceptOptions)
	if err != nil {
		log.ERROR.Println(err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "")

	_ = h.subscribe(r.Context(), conn)
}

func (h *SocketHub) subscribe(ctx context.Context, conn *websocket.Conn) error {
	ctx = conn.CloseRead(ctx)

	s := &socketSubscriber{
		send: make(chan []byte, 1024),
		closeSlow: func() {
			conn.Close(websocket.StatusPolicyViolation, "connection too slow to keep up with messages")
		},
	}

	h.addSubscriber(s)
	defer h.deleteSubscriber(s)

	// send welcome message
	h.register <- s

	for {
		select {
		case msg := <-s.send:
			if err := writeTimeout(ctx, socketWriteTimeout, conn, msg); err != nil {
				log.TRACE.Printf("ws write error: %v, len: %.1fkB, msg: %s", err, float64(len(msg))/1024, msg[:min(len(msg), 20)])
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}

// addSubscriber registers a subscriber.
func (h *SocketHub) addSubscriber(s *socketSubscriber) {
	h.mu.Lock()
	h.subscribers[s] = struct{}{}
	h.mu.Unlock()
}

// deleteSubscriber deletes the given subscriber.
func (h *SocketHub) deleteSubscriber(s *socketSubscriber) {
	h.mu.Lock()
	delete(h.subscribers, s)
	h.mu.Unlock()
}

func (h *SocketHub) welcome(subscriber *socketSubscriber, params []util.Param) {
	msg := make(map[string]json.RawMessage, len(params))
	sharders := make(map[string]util.Sharder)

	for _, p := range params {
		k := p.Key
		if p.Loadpoint != nil {
			k = "loadpoints." + p.UniqueID()
		}

		// Sharder values are split into shards and sent as a separate message
		if sharder, ok := (p.Val).(util.Sharder); ok {
			sharders[k] = sharder
		} else {
			msg[k] = json.RawMessage(socketEncode(p.Val))
		}
	}

	// send compact state first, then potentially large sharded data
	if b, err := json.Marshal(msg); err == nil {
		subscriber.send <- b
	}

	for k, sharder := range sharders {
		for key, val := range sharder.AllShards() {
			if b, err := json.Marshal(map[string]json.RawMessage{
				k + "." + key: json.RawMessage(socketEncode(val)),
			}); err == nil {
				subscriber.send <- b
			}
		}
	}
}

func (h *SocketHub) broadcast(p util.Param) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.subscribers) == 0 {
		return
	}

	msg := make(map[string]json.RawMessage)

	k := p.Key
	if p.Loadpoint != nil {
		k = "loadpoints." + p.UniqueID()
	}

	// Sharder splits data into chunks
	if sp, ok := (p.Val).(util.Sharder); ok {
		for key, val := range sp.ModifiedShards() {
			msg[k+"."+key] = json.RawMessage(socketEncode(val))
		}
	} else {
		msg[k] = json.RawMessage(socketEncode(p.Val))
	}

	b, _ := json.Marshal(msg)

	for s := range h.subscribers {
		select {
		case s.send <- b:
		default:
			s.closeSlow()
		}
	}
}

// Run starts data and status distribution
func (h *SocketHub) Run(in <-chan util.Param, cache *util.ParamCache) {
	for {
		select {
		case client := <-h.register:
			h.welcome(client, cache.All())
		case msg, ok := <-in:
			if !ok {
				return // break if channel closed
			}
			h.broadcast(msg)
		}
	}
}
