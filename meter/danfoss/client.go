package danfoss

import (
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"go.bug.st/serial"
)

// DefaultBaudrate is the RS485 baud rate used by all TLX variants.
const DefaultBaudrate = 19200

// DefaultTimeout is the per-request timeout applied to each read.
const DefaultTimeout = 2 * time.Second

// DefaultRetries is the number of additional attempts on top of the first
// request before returning an error.
const DefaultRetries = 2

const (
	maxFrameLength   = 128
	readChunkTimeout = 50 * time.Millisecond
)

// Config holds the parameters needed to open a ComLynx client.
type Config struct {
	// Device is the local serial port (e.g. /dev/ttyUSB0). Mutually
	// exclusive with URI.
	Device string
	// URI is a TCP endpoint exposing a raw RS485 bridge (host:port). Mutually
	// exclusive with Device.
	URI string
	// Baudrate defaults to 19200 when unset.
	Baudrate int
	// Timeout is the per-request timeout. Zero uses DefaultTimeout.
	Timeout time.Duration
	// Retries is the number of retries. Zero uses DefaultRetries.
	Retries int
	// Source is the address evcc advertises on the bus. Zero-value uses
	// DefaultSource.
	Source Address
	// Destination is the address of the inverter to query. Zero-value is
	// interpreted as "unset"; the caller is expected to run address
	// discovery before the first read.
	Destination Address
}

// Client is a single-inverter ComLynx client. All methods are safe for
// concurrent use.
type Client struct {
	mu      sync.Mutex
	cfg     Config
	log     *util.Logger
	t       transport
	dst     Address
	timeout time.Duration
	retries int
}

// transport abstracts over a serial port and a TCP connection.
type transport interface {
	io.ReadWriteCloser
	setReadTimeout(time.Duration) error
}

// New opens a connection and returns a ready Client. The caller may override
// the destination address via SetDestination before the first read.
func New(log *util.Logger, cfg Config) (*Client, error) {
	if cfg.Baudrate == 0 {
		cfg.Baudrate = DefaultBaudrate
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.Retries == 0 {
		cfg.Retries = DefaultRetries
	}
	if (cfg.Source == Address{}) {
		cfg.Source = DefaultSource
	}
	if cfg.Device == "" && cfg.URI == "" {
		return nil, errors.New("either device or uri must be set")
	}
	if cfg.Device != "" && cfg.URI != "" {
		return nil, errors.New("device and uri are mutually exclusive")
	}

	c := &Client{
		cfg:     cfg,
		log:     log,
		dst:     cfg.Destination,
		timeout: cfg.Timeout,
		retries: cfg.Retries,
	}
	if err := c.reopen(); err != nil {
		return nil, err
	}
	return c, nil
}

// Close releases the underlying transport.
func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.t == nil {
		return nil
	}
	err := c.t.Close()
	c.t = nil
	return err
}

// Destination returns the currently configured inverter address.
func (c *Client) Destination() Address {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.dst
}

// SetDestination updates the inverter address used by subsequent reads.
func (c *Client) SetDestination(dst Address) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.dst = dst
}

// Read queries a single TLX parameter by its 16-bit ID and returns the raw
// 32-bit value (little-endian, as reported by the inverter). Scaling into
// engineering units happens at the meter wrapper layer.
func (c *Client) Read(paramID uint16) (int32, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if (c.dst == Address{}) {
		return 0, errors.New("no destination address set")
	}

	req := encodeFrame(encodeMessage(c.cfg.Source, c.dst, typeCanRequest, encodeCanRequest(paramID)))

	var lastErr error
	for range c.retries {
		reply, err := c.exchange(req)
		if err == nil {
			msg, derr := decodeMessage(reply)
			if derr != nil {
				lastErr = derr
			} else if msg.Type != typeCanReply {
				// Accept only CanReply; refuse to parse unrelated frames.
				lastErr = fmt.Errorf("unexpected reply type 0x%02x", msg.Type)
			} else {
				v, verr := decodeCanReply(paramID, msg.Data)
				if verr == nil {
					return v, nil
				}
				lastErr = verr
			}
		} else {
			lastErr = err
			// Re-open on hard I/O errors; keep the socket otherwise.
			if isHardError(err) {
				_ = c.reopenLocked()
			}
		}
	}
	return 0, fmt.Errorf("read %04x: %w", paramID, lastErr)
}

// Ping issues a broadcast PingRequest and returns the source address of the
// first replying node, or every replying node if multi is true. Used by
// address discovery.
func (c *Client) Ping(multi bool) ([]Address, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	req := encodeFrame(encodeMessage(c.cfg.Source, Broadcast, typePingRequest, nil))
	if _, err := c.t.Write(req); err != nil {
		return nil, err
	}

	var addrs []Address
	deadline := time.Now().Add(c.timeout)
	for {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			break
		}
		frame, err := readFrame(c.t, remaining)
		if err != nil {
			if errors.Is(err, os.ErrDeadlineExceeded) {
				break
			}
			return addrs, err
		}
		body, err := decodeFrame(frame)
		if err != nil {
			c.log.TRACE.Printf("discard malformed frame: %v", err)
			continue
		}
		msg, err := decodeMessage(body)
		if err != nil {
			c.log.TRACE.Printf("discard undecodable reply: %v", err)
			continue
		}
		if msg.Type != typePingReply {
			continue
		}
		addrs = append(addrs, msg.Source)
		if !multi {
			break
		}
	}
	return addrs, nil
}

func (c *Client) exchange(req []byte) ([]byte, error) {
	if c.t == nil {
		if err := c.reopenLocked(); err != nil {
			return nil, err
		}
	}
	c.log.TRACE.Printf("tx: % x", req)
	if _, err := c.t.Write(req); err != nil {
		return nil, err
	}
	frame, err := readFrame(c.t, c.timeout)
	if err != nil {
		return nil, err
	}
	c.log.TRACE.Printf("rx: % x", frame)
	return decodeFrame(frame)
}

func (c *Client) reopen() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.reopenLocked()
}

func (c *Client) reopenLocked() error {
	if c.t != nil {
		_ = c.t.Close()
		c.t = nil
	}
	t, err := openTransport(c.cfg)
	if err != nil {
		return err
	}
	c.t = t
	return nil
}

func openTransport(cfg Config) (transport, error) {
	if cfg.Device != "" {
		p, err := serial.Open(cfg.Device, &serial.Mode{
			BaudRate: cfg.Baudrate,
			Parity:   serial.NoParity,
			DataBits: 8,
			StopBits: serial.OneStopBit,
		})
		if err != nil {
			return nil, fmt.Errorf("open serial %s: %w", cfg.Device, err)
		}
		if err := p.SetReadTimeout(readChunkTimeout); err != nil {
			_ = p.Close()
			return nil, err
		}
		return &serialTransport{Port: p}, nil
	}
	conn, err := net.DialTimeout("tcp", cfg.URI, cfg.Timeout)
	if err != nil {
		return nil, fmt.Errorf("dial %s: %w", cfg.URI, err)
	}
	return &tcpTransport{Conn: conn}, nil
}

type serialTransport struct {
	serial.Port
}

func (t *serialTransport) setReadTimeout(d time.Duration) error {
	return t.Port.SetReadTimeout(d)
}

type tcpTransport struct {
	net.Conn
}

func (t *tcpTransport) setReadTimeout(d time.Duration) error {
	return t.Conn.SetReadDeadline(time.Now().Add(d))
}

// readFrame reads bytes from r until it has a complete HDLC frame bounded by
// 0x7e flag sequences. Empty frames (flag-flag) are skipped.
func readFrame(r io.Reader, timeout time.Duration) ([]byte, error) {
	rt, _ := r.(interface{ setReadTimeout(time.Duration) error })

	buf := make([]byte, 0, 32)
	one := make([]byte, 1)
	started := false
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		if rt != nil {
			// Call before each read so TCP's SetReadDeadline gets refreshed;
			// serial ignores repeated calls with the same duration.
			_ = rt.setReadTimeout(readChunkTimeout)
		}
		n, err := r.Read(one)
		if err != nil {
			// A timeout on a single chunk read is fine; retry until the
			// overall deadline is exhausted.
			if errors.Is(err, os.ErrDeadlineExceeded) || isTimeoutErr(err) {
				continue
			}
			if n == 0 {
				return nil, err
			}
		}
		if n == 0 {
			continue
		}
		b := one[0]
		if !started {
			if b == flagSequence {
				buf = append(buf, b)
				started = true
			}
			continue
		}

		buf = append(buf, b)
		if b == flagSequence {
			if len(buf) == 2 {
				// Empty frame — treat the second flag as a fresh start.
				buf = buf[:1]
				continue
			}
			return buf, nil
		}
		if len(buf) > maxFrameLength {
			return nil, fmt.Errorf("frame exceeds %d bytes", maxFrameLength)
		}
	}
	return nil, os.ErrDeadlineExceeded
}

// isHardError reports whether an error should trigger a transport reopen.
func isHardError(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF) {
		return true
	}
	var netErr *net.OpError
	return errors.As(err, &netErr)
}

// isTimeoutErr reports whether the error is a transient timeout that should
// not abort the outer read loop.
func isTimeoutErr(err error) bool {
	if err == nil {
		return false
	}
	type timeoutError interface {
		Timeout() bool
	}
	var te timeoutError
	return errors.As(err, &te) && te.Timeout()
}
