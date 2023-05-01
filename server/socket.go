package server

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"nhooyr.io/websocket"
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

	// https://github.com/nhooyr/websocket/issues/218
	ua := strings.ToLower(r.Header.Get("User-Agent"))
	if strings.Contains(ua, "safari") && !strings.Contains(ua, "chrome") && !strings.Contains(ua, "android") {
		acceptOptions.CompressionMode = websocket.CompressionDisabled
	}

	conn, err := websocket.Accept(w, r, acceptOptions)
	if err != nil {
		log.ERROR.Println(err)
		return
	}
	defer conn.Close(websocket.StatusInternalError, "")

	err = h.subscribe(r.Context(), conn)

	if errors.Is(err, context.Canceled) {
		return
	}
	if cs := websocket.CloseStatus(err); cs == websocket.StatusNormalClosure || cs == websocket.StatusGoingAway {
		return
	}
	if err != nil {
		log.ERROR.Println(err)
		return
	}
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
	var msg strings.Builder
	msg.WriteString("{")
	for _, p := range params {
		if msg.Len() > 1 {
			msg.WriteString(",")
		}
		msg.WriteString(kv(p))
	}
	msg.WriteString("}")

	// should not block
	subscriber.send <- []byte(msg.String())
}

func (h *SocketHub) broadcast(p util.Param) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.subscribers) > 0 {
		msg := "{" + kv(p) + "}"

		for s := range h.subscribers {
			select {
			case s.send <- []byte(msg):
			default:
				s.closeSlow()
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
		case msg, ok := <-in:
			if !ok {
				return // break if channel closed
			}
			h.broadcast(msg)
		}
	}
}
