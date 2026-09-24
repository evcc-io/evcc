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
	reconnectDelay    = 10 * time.Second
)

// Connection is a persistent local connection to a Tuya device. Devices accept only a single local connection.
type Connection struct {
	log     *util.Logger
	addr    string
	id      string
	version string
	key     []byte

	mu    sync.Mutex
	conn  net.Conn
	codec *codec
	seq   uint32

	dps *util.Monitor[map[string]any]
}

// NewConnection creates a connection and starts it in the background
func NewConnection(ctx context.Context, log *util.Logger, host, id, key, version string) (*Connection, error) {
	if _, err := newCodec(version, []byte(key)); err != nil {
		return nil, err
	}

	c := &Connection{
		log:     log,
		addr:    util.DefaultPort(host, 6668),
		id:      id,
		version: version,
		key:     []byte(key),
		dps:     util.NewMonitor[map[string]any](readTimeout),
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

func (c *Connection) session(ctx context.Context) error {
	conn, err := (&net.Dialer{Timeout: ioTimeout}).DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return err
	}
	defer conn.Close()

	codec, _ := newCodec(c.version, c.key)
	r := bufio.NewReader(conn)

	c.mu.Lock()
	c.conn, c.codec, c.seq = conn, codec, 1
	c.mu.Unlock()

	defer func() {
		c.mu.Lock()
		c.conn, c.codec = nil, nil
		c.mu.Unlock()
	}()

	if c.version != "3.3" {
		if err := c.negotiate(conn, r, codec); err != nil {
			return fmt.Errorf("session key negotiation: %w", err)
		}
	}

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
		select {
		case <-ctx.Done():
			return nil
		case err := <-errC:
			return err
		case <-heartbeat.C:
			err = c.send(cmdHeartbeat, map[string]any{"gwId": c.id, "devId": c.id})
		case <-query.C:
			err = c.query()
		}

		if err != nil {
			return err
		}
	}
}

func (c *Connection) negotiate(conn net.Conn, r *bufio.Reader, codec *codec) error {
	localNonce := make([]byte, 16)
	if _, err := rand.Read(localNonce); err != nil {
		return err
	}

	if err := c.write(cmdSessKeyNegStart, localNonce); err != nil {
		return err
	}

	if err := conn.SetReadDeadline(time.Now().Add(ioTimeout)); err != nil {
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

	if err := c.write(cmdSessKeyNegFinish, hmacSum(c.key, remoteNonce)); err != nil {
		return err
	}

	sessionKey, err := codec.deriveSessionKey(localNonce, remoteNonce)
	if err != nil {
		return err
	}

	c.mu.Lock()
	codec.sessionKey = sessionKey
	c.mu.Unlock()

	return nil
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

		c.log.TRACE.Printf("recv: cmd %d: %s", msg.cmd, msg.payload)

		if msg.cmd == cmdHeartbeat {
			// device is alive, keep last values valid
			select {
			case <-c.dps.Done():
				c.dps.SetFunc(func(v map[string]any) map[string]any { return v })
			default:
			}
			continue
		}

		if len(msg.payload) == 0 {
			continue
		}

		var res struct {
			Dps  map[string]any `json:"dps"`
			Data struct {
				Dps map[string]any `json:"dps"`
			} `json:"data"`
		}

		if err := json.Unmarshal(msg.payload, &res); err != nil {
			c.log.DEBUG.Printf("invalid payload: %s", msg.payload)
			continue
		}

		dps := res.Dps
		if dps == nil {
			dps = res.Data.Dps
		}

		if dps != nil {
			c.dps.SetFunc(func(v map[string]any) map[string]any {
				res := maps.Clone(v)
				if res == nil {
					res = make(map[string]any)
				}
				maps.Copy(res, dps)
				return res
			})
		}
	}
}

func (c *Connection) query() error {
	if c.version == "3.3" {
		return c.send(cmdDpQuery, map[string]any{"gwId": c.id, "devId": c.id, "uid": c.id, "t": c.timestamp()})
	}
	return c.send(cmdDpQueryNew, map[string]any{})
}

func (c *Connection) timestamp() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

func (c *Connection) send(cmd uint32, data map[string]any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return c.write(cmd, b)
}

func (c *Connection) write(cmd uint32, payload []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return errors.New("not connected")
	}

	c.log.TRACE.Printf("send: cmd %d: %s", cmd, payload)

	frame, err := c.codec.encode(c.seq, cmd, payload)
	if err != nil {
		return err
	}
	c.seq++

	if err := c.conn.SetWriteDeadline(time.Now().Add(ioTimeout)); err != nil {
		return err
	}

	_, err = c.conn.Write(frame)
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

// Set writes data points
func (c *Connection) Set(dps map[string]any) error {
	if c.version == "3.3" {
		return c.send(cmdControl, map[string]any{"devId": c.id, "uid": c.id, "t": c.timestamp(), "dps": dps})
	}
	return c.send(cmdControlNew, map[string]any{"protocol": 5, "t": time.Now().Unix(), "data": map[string]any{"dps": dps}})
}
