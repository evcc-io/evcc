package tuya

import (
	"bufio"
	"context"
	"crypto/hmac"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
)

const (
	heartbeatInterval = 10 * time.Second
	queryInterval     = time.Minute
	readTimeout       = 3 * heartbeatInterval
	ioTimeout         = 10 * time.Second
	handshakeTimeout  = 5 * time.Second
	reconnectDelay    = 10 * time.Second
)

// versions in order of detection
var versions = []string{"3.5", "3.4", "3.3"}

// Connection is a persistent local connection to a Tuya device. Devices accept only a single local connection.
// The protocol version is detected on first connect.
type Connection struct {
	log  *util.Logger
	addr string
	id   string
	key  []byte

	version string // detected version, only accessed by run

	mu    sync.Mutex
	conn  net.Conn
	codec *codec
	seq   uint32

	dps *util.Monitor[map[string]any]
}

// NewConnection creates a connection and starts it in the background
func NewConnection(ctx context.Context, log *util.Logger, host, id, key string) (*Connection, error) {
	if len(key) != 16 {
		return nil, fmt.Errorf("invalid local key length: %d", len(key))
	}

	c := &Connection{
		log:  log,
		addr: util.DefaultPort(host, 6668),
		id:   id,
		key:  []byte(key),
		dps:  util.NewMonitor[map[string]any](readTimeout),
	}

	go c.run(ctx)

	return c, nil
}

func (c *Connection) run(ctx context.Context) {
	for {
		if err := c.session(ctx); err != nil && ctx.Err() == nil {
			c.log.ERROR.Println(err)
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(reconnectDelay):
		}
	}
}

// session connects using the detected version, or tries all versions if none has been detected yet
func (c *Connection) session(ctx context.Context) error {
	candidates := versions
	if c.version != "" {
		candidates = []string{c.version}
	}

	var errs error
	for _, version := range candidates {
		conn, err := (&net.Dialer{Timeout: ioTimeout}).DialContext(ctx, "tcp", c.addr)
		if err != nil {
			return err
		}

		codec, _ := newCodec(version, c.key)
		r := bufio.NewReader(conn)

		seq, err := c.handshake(conn, r, codec)
		if err != nil {
			conn.Close()
			errs = errors.Join(errs, fmt.Errorf("%s: %w", version, err))
			continue
		}

		if c.version == "" {
			c.log.DEBUG.Println("detected protocol version", version)
			c.version = version
		}

		return c.serve(ctx, conn, r, codec, seq)
	}

	// device may have been updated to a different version
	c.version = ""

	return fmt.Errorf("handshake failed, check local key: %w", errs)
}

// handshake verifies protocol version and local key, negotiating the session key if required.
// It returns the next sequence number.
func (c *Connection) handshake(conn net.Conn, r *bufio.Reader, codec *codec) (uint32, error) {
	seq := uint32(1)

	if err := conn.SetDeadline(time.Now().Add(handshakeTimeout)); err != nil {
		return seq, err
	}

	if codec.version != "3.3" {
		return seq, c.negotiate(conn, r, codec, &seq)
	}

	// 3.3 has no handshake, verify by decoding the status response
	cmd, data := c.queryMsg(codec.version)
	if err := c.writeTo(conn, codec, &seq, cmd, data); err != nil {
		return seq, err
	}

	frame, err := readFrame(r)
	if err != nil {
		return seq, err
	}

	msg, err := codec.decode(frame)
	if err != nil {
		return seq, err
	}

	if !json.Valid(msg.payload) {
		return seq, errors.New("invalid response")
	}

	c.handle(msg)

	return seq, nil
}

func (c *Connection) negotiate(conn net.Conn, r *bufio.Reader, codec *codec, seq *uint32) error {
	localNonce := make([]byte, 16)
	if _, err := rand.Read(localNonce); err != nil {
		return err
	}

	if err := c.writeRaw(conn, codec, seq, cmdSessKeyNegStart, localNonce); err != nil {
		return err
	}

	frame, err := readFrame(r)
	if err != nil {
		return err
	}

	msg, err := codec.decode(frame)
	if err != nil {
		return err
	}

	if msg.cmd != cmdSessKeyNegResp || len(msg.payload) < 48 {
		return fmt.Errorf("unexpected response: cmd %d, length %d", msg.cmd, len(msg.payload))
	}

	remoteNonce := msg.payload[:16]
	if !hmac.Equal(hmacSum(c.key, localNonce), msg.payload[16:48]) {
		return errors.New("invalid local key")
	}

	if err := c.writeRaw(conn, codec, seq, cmdSessKeyNegFinish, hmacSum(c.key, remoteNonce)); err != nil {
		return err
	}

	codec.sessionKey, err = codec.deriveSessionKey(localNonce, remoteNonce)
	return err
}

func (c *Connection) serve(ctx context.Context, conn net.Conn, r *bufio.Reader, codec *codec, seq uint32) error {
	defer conn.Close()

	if err := conn.SetDeadline(time.Time{}); err != nil {
		return err
	}

	c.mu.Lock()
	c.conn, c.codec, c.seq = conn, codec, seq
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.conn, c.codec = nil, nil
		c.mu.Unlock()
	}()

	errC := make(chan error, 1)
	go func() { errC <- c.receive(conn, r, codec) }()

	if err := c.query(); err != nil {
		return err
	}

	heartbeat := time.NewTicker(heartbeatInterval)
	defer heartbeat.Stop()

	query := time.NewTicker(queryInterval)
	defer query.Stop()

	for {
		var err error

		select {
		case <-ctx.Done():
			return nil
		case err := <-errC:
			return err
		case <-heartbeat.C:
			err = c.send(func(string) (uint32, any) {
				return cmdHeartbeat, map[string]any{"gwId": c.id, "devId": c.id}
			})
		case <-query.C:
			err = c.query()
		}

		if err != nil {
			return err
		}
	}
}

func (c *Connection) receive(conn net.Conn, r *bufio.Reader, codec *codec) error {
	for {
		if err := conn.SetReadDeadline(time.Now().Add(readTimeout)); err != nil {
			return err
		}

		frame, err := readFrame(r)
		if err != nil {
			return err
		}

		msg, err := codec.decode(frame)
		if err != nil {
			c.log.DEBUG.Printf("invalid frame: %v", err)
			continue
		}

		c.handle(msg)
	}
}

func (c *Connection) handle(msg message) {
	c.log.TRACE.Printf("recv: cmd %d: %s", msg.cmd, msg.payload)

	if msg.cmd == cmdHeartbeat {
		// device is alive, keep last values valid
		select {
		case <-c.dps.Done():
			c.dps.SetFunc(func(v map[string]any) map[string]any { return v })
		default:
		}
		return
	}

	if len(msg.payload) == 0 {
		return
	}

	var res struct {
		Dps  map[string]any `json:"dps"`
		Data struct {
			Dps map[string]any `json:"dps"`
		} `json:"data"`
	}

	if err := json.Unmarshal(msg.payload, &res); err != nil {
		c.log.DEBUG.Printf("invalid payload: %s", msg.payload)
		return
	}

	dps := res.Dps
	if dps == nil {
		dps = res.Data.Dps
	}

	if dps != nil {
		c.merge(dps)
	}
}

func (c *Connection) timestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func (c *Connection) queryMsg(version string) (uint32, any) {
	if version == "3.3" {
		return cmdDpQuery, map[string]any{"gwId": c.id, "devId": c.id, "uid": c.id, "t": c.timestamp()}
	}
	return cmdDpQueryNew, map[string]any{}
}

func (c *Connection) controlMsg(version string, dps map[string]any) (uint32, any) {
	if version == "3.3" {
		return cmdControl, map[string]any{"devId": c.id, "uid": c.id, "t": c.timestamp(), "dps": dps}
	}
	return cmdControlNew, map[string]any{"protocol": 5, "t": time.Now().Unix(), "data": map[string]any{"dps": dps}}
}

func (c *Connection) query() error {
	return c.send(c.queryMsg)
}

// send writes a message built for the protocol version of the current connection
func (c *Connection) send(msg func(version string) (uint32, any)) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return errors.New("not connected")
	}

	cmd, data := msg(c.codec.version)
	return c.writeTo(c.conn, c.codec, &c.seq, cmd, data)
}

// writeTo marshals and writes a message
func (c *Connection) writeTo(conn net.Conn, codec *codec, seq *uint32, cmd uint32, data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.writeRaw(conn, codec, seq, cmd, b)
}

func (c *Connection) writeRaw(conn net.Conn, codec *codec, seq *uint32, cmd uint32, payload []byte) error {
	c.log.TRACE.Printf("send: cmd %d: %s", cmd, payload)

	frame, err := codec.encode(*seq, cmd, payload)
	if err != nil {
		return err
	}
	*seq++

	if err := conn.SetWriteDeadline(time.Now().Add(ioTimeout)); err != nil {
		return err
	}

	_, err = conn.Write(frame)
	return err
}

// Dps returns the current data points
func (c *Connection) Dps() (map[string]any, error) {
	var res map[string]any
	err := c.dps.GetFunc(func(v map[string]any) {
		res = maps.Clone(v)
	})
	return res, err
}

// DpsContext returns the current data points, waiting for the first update until the context is done
func (c *Connection) DpsContext(ctx context.Context) (map[string]any, error) {
	var res map[string]any
	err := c.dps.GetFuncContext(ctx, func(v map[string]any) {
		res = maps.Clone(v)
	})
	return res, err
}

// Set writes data points. Written values are cached until the device reports them.
func (c *Connection) Set(dps map[string]any) error {
	if err := c.send(func(version string) (uint32, any) {
		return c.controlMsg(version, dps)
	}); err != nil {
		return err
	}

	// normalize to json types as received from the device
	b, err := json.Marshal(dps)
	if err != nil {
		return err
	}

	var res map[string]any
	if err := json.Unmarshal(b, &res); err != nil {
		return err
	}

	select {
	case <-c.dps.Done():
		c.merge(res)
	default:
	}

	return nil
}

func (c *Connection) merge(dps map[string]any) {
	c.dps.SetFunc(func(v map[string]any) map[string]any {
		res := maps.Clone(v)
		if res == nil {
			res = make(map[string]any)
		}
		maps.Copy(res, dps)
		return res
	})
}
