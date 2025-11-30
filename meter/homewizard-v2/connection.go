package v2

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/transport"
)

const (
	retryDelay   = 5 * time.Second
	authTimeout  = 40 * time.Second
	writeTimeout = 10 * time.Second
)

// MessageHandler is called when a message is received
type MessageHandler func(msgType string, data json.RawMessage) error

// Connection manages a WebSocket connection to a HomeWizard device
type Connection struct {
	log      *util.Logger
	host     string
	token    string
	handler  MessageHandler
	topics   []string
	conn     *websocket.Conn
	connMu   sync.RWMutex
	writeMu  sync.Mutex
	stopC    chan struct{}
	stoppedC chan struct{}
}

// NewConnection creates a new WebSocket connection manager
// If no topics are provided, defaults to ["measurement"]
func NewConnection(host, token string, handler MessageHandler, topics ...string) *Connection {
	log := util.NewLogger("homewizard-v2").Redact(token)

	// Default to measurement topic if none specified
	if len(topics) == 0 {
		topics = []string{"measurement"}
	}

	return &Connection{
		log:      log,
		host:     host,
		token:    token,
		handler:  handler,
		topics:   topics,
		stopC:    make(chan struct{}),
		stoppedC: make(chan struct{}),
	}
}

// Start begins the connection lifecycle in the background
func (c *Connection) Start(errC chan error) {
	go c.run(errC)
}

// Stop gracefully closes the connection
func (c *Connection) Stop() {
	close(c.stopC)
	<-c.stoppedC
}

func (c *Connection) run(errC chan error) {
	var once sync.Once
	defer close(c.stoppedC)

	for {
		select {
		case <-c.stopC:
			c.closeConn()
			return
		default:
		}

		if err := c.connect(); err != nil {
			// handle initial connection error immediately
			once.Do(func() {
				if errC != nil {
					errC <- err
				}
			})

			c.log.ERROR.Println(err)

			select {
			case <-c.stopC:
				return
			case <-time.After(retryDelay):
				continue
			}
		}

		// Signal successful connection on first attempt
		once.Do(func() {
			if errC != nil {
				close(errC)
			}
		})

		// Read loop
		c.readLoop()
	}
}

func (c *Connection) connect() error {
	uri := fmt.Sprintf("wss://%s/api/ws", c.host)

	// Prepare dial options with insecure transport
	opts := &websocket.DialOptions{
		HTTPClient: &http.Client{
			Transport: transport.Insecure(),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), request.Timeout)
	defer cancel()

	conn, _, err := websocket.Dial(ctx, uri, opts)
	if err != nil {
		return fmt.Errorf("dial: %w", err)
	}

	c.connMu.Lock()
	c.conn = conn
	c.connMu.Unlock()

	// Perform authentication handshake
	if err := c.authenticate(); err != nil {
		_ = c.closeConn()
		return fmt.Errorf("auth: %w", err)
	}

	// Subscribe to configured topics
	for _, topic := range c.topics {
		if err := c.subscribe(topic); err != nil {
			_ = c.closeConn()
			return fmt.Errorf("subscribe: %w", err)
		}
	}

	return nil
}

func (c *Connection) authenticate() error {
	ctx, cancel := context.WithTimeout(context.Background(), authTimeout)
	defer cancel()

	// Wait for authorization_requested message
	var authReq AuthRequest
	if err := c.readMessage(ctx, &authReq); err != nil {
		return fmt.Errorf("waiting for auth request: %w", err)
	}

	if authReq.Type != "authorization_requested" {
		return fmt.Errorf("unexpected message type: %s, expected: authorization_requested", authReq.Type)
	}

	c.log.TRACE.Printf("received auth request, api_version: %s", authReq.Data.APIVersion)

	// Send authorization with token
	authResp := AuthResponse{
		Type: "authorization",
		Data: c.token,
	}

	if err := c.writeMessage(ctx, authResp); err != nil {
		return fmt.Errorf("sending auth: %w", err)
	}

	// Wait for authorization confirmation
	var authConfirm AuthConfirm
	if err := c.readMessage(ctx, &authConfirm); err != nil {
		return fmt.Errorf("waiting for auth confirm: %w", err)
	}

	if authConfirm.Type != "authorized" {
		return fmt.Errorf("unexpected message type: %s, expected: authorized", authConfirm.Type)
	}

	c.log.DEBUG.Println("authenticated successfully")

	return nil
}

func (c *Connection) subscribe(topic string) error {
	sub := Subscribe{
		Type: "subscribe",
		Data: topic,
	}

	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()

	if err := c.writeMessage(ctx, sub); err != nil {
		return fmt.Errorf("subscribing to %s: %w", topic, err)
	}

	c.log.DEBUG.Printf("subscribed to topic: %s", topic)

	return nil
}

func (c *Connection) readLoop() {
	for {
		select {
		case <-c.stopC:
			return
		default:
		}

		c.connMu.RLock()
		conn := c.conn
		c.connMu.RUnlock()

		if conn == nil {
			return
		}

		_, b, err := conn.Read(context.Background())
		if err != nil {
			c.log.TRACE.Println("read:", err)
			_ = c.closeConn()
			return
		}

		c.log.TRACE.Printf("recv: %s", b)

		// Parse base message to get type
		var msg Message
		if err := json.Unmarshal(b, &msg); err != nil {
			c.log.ERROR.Printf("parse message: %v", err)
			continue
		}

		// Handle errors
		if msg.Type == "error" {
			var errMsg ErrorMessage
			if err := json.Unmarshal(b, &errMsg); err == nil {
				c.log.ERROR.Printf("server error: %s", errMsg.Data.Message)
			}
			continue
		}

		// Route to handler
		if c.handler != nil {
			if err := c.handler(msg.Type, msg.Data); err != nil {
				c.log.ERROR.Printf("handle message type %s: %v", msg.Type, err)
			}
		}
	}
}

func (c *Connection) readMessage(ctx context.Context, v any) error {
	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return api.ErrTimeout
	}

	_, b, err := conn.Read(ctx)
	if err != nil {
		return err
	}

	c.log.TRACE.Printf("recv: %s", b)

	return json.Unmarshal(b, v)
}

func (c *Connection) writeMessage(ctx context.Context, v any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.connMu.RLock()
	conn := c.conn
	c.connMu.RUnlock()

	if conn == nil {
		return api.ErrTimeout
	}

	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	c.log.TRACE.Printf("send: %s", b)

	return conn.Write(ctx, websocket.MessageText, b)
}

// Send sends a message over the WebSocket connection (for battery control, etc.)
func (c *Connection) Send(v any) error {
	ctx, cancel := context.WithTimeout(context.Background(), writeTimeout)
	defer cancel()

	return c.writeMessage(ctx, v)
}

func (c *Connection) closeConn() error {
	c.connMu.Lock()
	defer c.connMu.Unlock()

	if c.conn == nil {
		return nil
	}

	err := c.conn.Close(websocket.StatusNormalClosure, "")
	c.conn = nil
	return err
}
