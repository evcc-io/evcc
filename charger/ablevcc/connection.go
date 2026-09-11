package ablevcc

// LICENSE

// Copyright (c) evcc.io (andig, naltatis, premultiply)

// This module is NOT covered by the MIT license. All rights reserved.

// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.

// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"go.bug.st/serial"
)

const (
	// Baudrate is the fixed EVCC line speed
	Baudrate = 38400

	// Timeout is the default reply timeout
	Timeout = 2 * time.Second
)

// ErrRejected indicates that the device did not accept the command
var ErrRejected = errors.New("command rejected")

var (
	mu        sync.Mutex
	instances = make(map[string]*Connection)
)

// Connection is a shared RS485 bus connection. Since all modules on the bus answer
// on the same wire, transactions are serialized across all chargers using it.
type Connection struct {
	mu      sync.Mutex
	log     *util.Logger
	dial    func() (port, error)
	timeout time.Duration
	conn    port
	r       *bufio.Reader
	closed  bool
	refs    int // number of chargers using the connection, guarded by the package mutex
}

// Instance returns the shared connection for the given bus. Chargers on the same
// device or uri share a single physical connection.
func Instance(ctx context.Context, log *util.Logger, device, uri string, timeout time.Duration) (*Connection, error) {
	if (device == "") == (uri == "") {
		return nil, errors.New("can only have either uri or device")
	}

	if timeout <= 0 {
		timeout = Timeout
	}

	key := "uri:" + uri
	if device != "" {
		key = "device:" + device
	}

	mu.Lock()
	defer mu.Unlock()

	conn, ok := instances[key]
	if !ok {
		conn = &Connection{
			log:     log,
			timeout: timeout,
			dial:    dialer(device, uri, timeout),
		}

		instances[key] = conn
	}

	conn.refs++

	// release the connection once the last charger using it is gone
	go func() {
		<-ctx.Done()
		release(key, conn)
	}()

	return conn, nil
}

func release(key string, conn *Connection) {
	mu.Lock()
	defer mu.Unlock()

	conn.refs--
	if conn.refs > 0 {
		return
	}

	// the key may already refer to a newer connection
	if instances[key] == conn {
		delete(instances, key)
	}

	conn.mu.Lock()
	defer conn.mu.Unlock()

	// prevent late callers from reopening the port
	conn.closed = true
	conn.close()
}

// port is the transport with net.Conn deadline semantics
type port interface {
	io.ReadWriteCloser
	SetReadDeadline(t time.Time) error
}

// dialer creates the transport specific connect function
func dialer(device, uri string, timeout time.Duration) func() (port, error) {
	if device != "" {
		return func() (port, error) {
			p, err := serial.Open(device, &serial.Mode{
				BaudRate: Baudrate,
				DataBits: 8,
				Parity:   serial.NoParity,
				StopBits: serial.OneStopBit,
			})
			if err != nil {
				return nil, err
			}

			return &serialPort{p}, nil
		}
	}

	return func() (port, error) {
		return net.DialTimeout("tcp", uri, timeout)
	}
}

// serialPort adapts go.bug.st/serial to net.Conn deadline semantics
type serialPort struct {
	serial.Port
}

func (p *serialPort) SetReadDeadline(t time.Time) error {
	return p.Port.SetReadTimeout(max(time.Until(t), 0))
}

// Read translates the serial read timeout convention into an error. The port
// signals a timeout as (0, nil) which would make bufio spin until it gives up
// with io.ErrNoProgress.
func (p *serialPort) Read(b []byte) (int, error) {
	n, err := p.Port.Read(b)
	if n == 0 && err == nil {
		return 0, os.ErrDeadlineExceeded
	}

	return n, err
}

// Transact sends a command to the given module address and returns the reply data.
// Data is returned verbatim since the firmware reply is not numeric.
func (c *Connection) Transact(addr, cmd uint8, payload string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	var res string
	var err error

	// retry once on communication error, but never on a rejected command
	for range 2 {
		if res, err = c.transact(addr, cmd, payload); err == nil || errors.Is(err, ErrRejected) {
			break
		}

		// drop the connection so that the next attempt reconnects
		c.close()
	}

	return res, err
}

// Int sends a command and converts the numeric reply
func (c *Connection) Int(addr, cmd uint8, payload string) (int, error) {
	s, err := c.Transact(addr, cmd, payload)
	if err != nil {
		return 0, err
	}

	res, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("invalid data: %q", s)
	}

	return res, nil
}

func (c *Connection) transact(addr, cmd uint8, payload string) (string, error) {
	if err := c.connect(); err != nil {
		return "", err
	}

	req := fmt.Sprintf("!%d %02d\r\n", addr, cmd)
	if payload != "" {
		req = fmt.Sprintf("!%d %02d %s\r\n", addr, cmd, payload)
	}

	// discard leftover replies of previous transactions
	c.r.Reset(c.conn)

	c.log.TRACE.Printf("send: %q", req)

	if _, err := c.conn.Write([]byte(req)); err != nil {
		return "", err
	}

	deadline := time.Now().Add(c.timeout)

	// modules of other addresses may answer on the same bus
	for time.Now().Before(deadline) {
		if err := c.conn.SetReadDeadline(deadline); err != nil {
			return "", err
		}

		line, err := c.r.ReadString('\n')
		if err != nil {
			return "", err
		}

		c.log.TRACE.Printf("recv: %q", line)

		resAddr, resCmd, data, err := parse(line)
		if err != nil || resAddr != addr || resCmd != cmd {
			continue
		}

		if data == "ERR" {
			return "", fmt.Errorf("%w: %02d", ErrRejected, cmd)
		}

		return data, nil
	}

	return "", os.ErrDeadlineExceeded
}

func (c *Connection) connect() error {
	if c.closed {
		return net.ErrClosed
	}

	if c.conn != nil {
		return nil
	}

	conn, err := c.dial()
	if err != nil {
		return err
	}

	c.conn = conn
	c.r = bufio.NewReader(conn)

	return nil
}

func (c *Connection) close() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
		c.r = nil
	}
}

// parse splits a device reply of the form ">n cc[ data]". The device is known to
// pad replies with additional spaces.
func parse(line string) (uint8, uint8, string, error) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(line), ">")
	if !ok {
		return 0, 0, "", fmt.Errorf("invalid reply: %q", line)
	}

	f := strings.Fields(rest)
	if len(f) < 2 || len(f) > 3 {
		return 0, 0, "", fmt.Errorf("invalid reply: %q", line)
	}

	addr, err := strconv.ParseUint(f[0], 10, 8)
	if err != nil {
		return 0, 0, "", fmt.Errorf("invalid address: %q", line)
	}

	cmd, err := strconv.ParseUint(f[1], 10, 8)
	if err != nil {
		return 0, 0, "", fmt.Errorf("invalid command: %q", line)
	}

	var data string
	if len(f) == 3 {
		data = f[2]
	}

	return uint8(addr), uint8(cmd), data, nil
}
