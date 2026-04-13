package danfoss

import (
	"net"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fakeTransport implements transport backed by a net.Pipe so no real hardware
// is needed.
type fakeTransport struct {
	net.Conn
}

func (f *fakeTransport) setReadTimeout(d time.Duration) error {
	return f.Conn.SetReadDeadline(time.Now().Add(d))
}

// newFakeClient returns a Client whose transport is one end of a net.Pipe, and
// the other end for the test to write inverter replies into.
func newFakeClient(t *testing.T, dst Address) (*Client, net.Conn) {
	t.Helper()
	local, remote := net.Pipe()
	log := util.NewLogger("danfoss-test")
	c := &Client{
		cfg:     Config{Source: DefaultSource, Timeout: 500 * time.Millisecond, Retries: 0},
		log:     log,
		t:       &fakeTransport{Conn: local},
		dst:     dst,
		timeout: 500 * time.Millisecond,
		retries: 0,
	}
	return c, remote
}

// buildCanReplyFrame builds a complete HDLC frame for a CanReply with the
// given paramID and raw value (little-endian int32).
func buildCanReplyFrame(src, dst Address, paramID uint16, value int32) []byte {
	data := []byte{
		0xc8, 0x0d, 0x80,
		byte(paramID >> 8), byte(paramID & 0xff),
		0x07, // UNSIGNED_32
		byte(value), byte(value >> 8), byte(value >> 16), byte(value >> 24),
	}
	body := encodeMessage(src, dst, typeCanReply, data)
	return encodeFrame(body)
}

func TestClientRead(t *testing.T) {
	invAddr := Address{Network: 0xc, Subnet: 0x6, Node: 0xb1}
	c, remote := newFakeClient(t, invAddr)
	defer c.Close()
	defer remote.Close()

	// The client will send a CanRequest; we respond with a CanReply.
	go func() {
		// Drain the request.
		buf := make([]byte, 128)
		if _, err := remote.Read(buf); err != nil {
			return
		}
		// Write a reply: GridPowerTotal = 8500 W
		reply := buildCanReplyFrame(invAddr, DefaultSource, paramGridPowerTotal, 8500)
		_, _ = remote.Write(reply)
	}()

	v, err := c.Read(paramGridPowerTotal)
	require.NoError(t, err)
	assert.EqualValues(t, 8500, v)
}

func TestClientReadTimeout(t *testing.T) {
	invAddr := Address{Network: 0xc, Subnet: 0x6, Node: 0xb1}
	c, remote := newFakeClient(t, invAddr)
	defer c.Close()
	defer remote.Close()

	// No reply — the client should time out.
	go func() {
		buf := make([]byte, 128)
		_, _ = remote.Read(buf)
		// Don't write anything back.
	}()

	_, err := c.Read(paramGridPowerTotal)
	assert.Error(t, err, "expected error when inverter does not reply")
}

func TestClientReadParamMismatch(t *testing.T) {
	invAddr := Address{Network: 0xc, Subnet: 0x6, Node: 0xb1}
	c, remote := newFakeClient(t, invAddr)
	defer c.Close()
	defer remote.Close()

	go func() {
		buf := make([]byte, 128)
		if _, err := remote.Read(buf); err != nil {
			return
		}
		// Reply for the WRONG parameter.
		reply := buildCanReplyFrame(invAddr, DefaultSource, paramTotalEnergy, 12345)
		_, _ = remote.Write(reply)
	}()

	_, err := c.Read(paramGridPowerTotal)
	assert.ErrorContains(t, err, "unexpected parameter")
}
