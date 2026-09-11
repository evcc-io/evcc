package ablevcc

import (
	"bufio"
	"context"
	"io"
	"net"
	"testing"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testConnection wires a Connection to an in-memory device answering with replies.
// The device answers each request with the corresponding entry, the last entry is
// repeated for further requests. An empty entry means no answer at all.
func testConnection(t *testing.T, replies ...string) *Connection {
	t.Helper()

	client, device := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = device.Close()
	})

	go func() {
		r := bufio.NewReader(device)

		for i := 0; ; i++ {
			if _, err := r.ReadString('\n'); err != nil {
				return
			}

			reply := replies[min(i, len(replies)-1)]
			if reply == "" {
				continue
			}

			if _, err := io.WriteString(device, reply); err != nil {
				return
			}
		}
	}()

	return &Connection{
		log:     util.NewLogger("test"),
		timeout: 100 * time.Millisecond,
		dial: func() (port, error) {
			return client, nil
		},
	}
}

func TestTransact(t *testing.T) {
	c := testConnection(t, ">1 02 0005\r\n")

	res, err := c.Transact(1, 2, "")
	require.NoError(t, err)
	assert.Equal(t, "0005", res)
}

func TestTransactExcessSpaces(t *testing.T) {
	c := testConnection(t, ">1  02   0005 \r\n")

	res, err := c.Transact(1, 2, "")
	require.NoError(t, err)
	assert.Equal(t, "0005", res)
}

func TestTransactAck(t *testing.T) {
	c := testConnection(t, ">1 12\r\n")

	res, err := c.Transact(1, 12, "0267")
	require.NoError(t, err)
	assert.Empty(t, res)
}

// replies of other modules on the same bus must be skipped
func TestTransactForeignAddress(t *testing.T) {
	c := testConnection(t, ">2 02 0000\r\n>1 02 0005\r\n")

	res, err := c.Transact(1, 2, "")
	require.NoError(t, err)
	assert.Equal(t, "0005", res)
}

func TestTransactRejected(t *testing.T) {
	c := testConnection(t, ">1 12 ERR\r\n")

	_, err := c.Transact(1, 12, "0999")
	assert.ErrorIs(t, err, ErrRejected)
}

func TestTransactTimeout(t *testing.T) {
	c := testConnection(t, "")

	_, err := c.Transact(1, 2, "")
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrRejected)
}

// a late reply of another module must not restart the timeout
func TestTransactDeadline(t *testing.T) {
	client, device := net.Pipe()
	t.Cleanup(func() {
		_ = client.Close()
		_ = device.Close()
	})

	go func() {
		r := bufio.NewReader(device)
		if _, err := r.ReadString('\n'); err != nil {
			return
		}

		time.Sleep(150 * time.Millisecond)
		_, _ = io.WriteString(device, ">2 02 0000\r\n")
	}()

	c := &Connection{
		log:     util.NewLogger("test"),
		timeout: 200 * time.Millisecond,
		dial: func() (port, error) {
			return client, nil
		},
	}

	start := time.Now()
	_, err := c.Transact(1, 2, "")
	require.Error(t, err)
	assert.Less(t, time.Since(start), 300*time.Millisecond)
}

func TestTransactFirmware(t *testing.T) {
	c := testConnection(t, ">1 01 V2.3\r\n")

	res, err := c.Transact(1, 1, "")
	require.NoError(t, err)
	assert.Equal(t, "V2.3", res)
}

func TestInt(t *testing.T) {
	c := testConnection(t, ">1 26 0267\r\n")

	res, err := c.Int(1, 26, "")
	require.NoError(t, err)
	assert.Equal(t, 267, res)
}

func TestParse(t *testing.T) {
	for _, tc := range []struct {
		line string
		addr uint8
		cmd  uint8
		data string
		err  bool
	}{
		{">1 02 0005\r\n", 1, 2, "0005", false},
		{">1 02\r\n", 1, 2, "", false},
		{">8  11   0850  \r\n", 8, 11, "0850", false},
		{">1 01 V2.3\r\n", 1, 1, "V2.3", false},
		{">1 12 ERR\r\n", 1, 12, "ERR", false},
		{"!1 02 0005\r\n", 0, 0, "", true}, // request echo
		{">1\r\n", 0, 0, "", true},
		{">1 02 0005 0006\r\n", 0, 0, "", true},
		{">x 02\r\n", 0, 0, "", true},
		{"", 0, 0, "", true},
	} {
		addr, cmd, data, err := parse(tc.line)

		if tc.err {
			assert.Error(t, err, tc.line)
			continue
		}

		require.NoError(t, err, tc.line)
		assert.Equal(t, tc.addr, addr, tc.line)
		assert.Equal(t, tc.cmd, cmd, tc.line)
		assert.Equal(t, tc.data, data, tc.line)
	}
}

func TestInstanceRequiresSingleTransport(t *testing.T) {
	log := util.NewLogger("test")

	_, err := Instance(t.Context(), log, "", "", 0)
	assert.Error(t, err)

	_, err = Instance(t.Context(), log, "/dev/ttyUSB0", "localhost:4196", 0)
	assert.Error(t, err)
}

func TestInstanceShared(t *testing.T) {
	log := util.NewLogger("test")

	c1, err := Instance(t.Context(), log, "", "localhost:14196", 0)
	require.NoError(t, err)

	c2, err := Instance(t.Context(), log, "", "localhost:14196", 0)
	require.NoError(t, err)

	assert.Same(t, c1, c2, "chargers on the same bus must share the connection")
}

func TestInstanceDistinctTransports(t *testing.T) {
	log := util.NewLogger("test")

	c1, err := Instance(t.Context(), log, "foo", "", 0)
	require.NoError(t, err)

	c2, err := Instance(t.Context(), log, "", "foo", 0)
	require.NoError(t, err)

	assert.NotSame(t, c1, c2, "device and uri of the same name must not share the connection")
}

// a cancelled context must remove the connection from the pool so that a
// recreated charger (config reload, failed constructor, device test) gets a fresh one
func TestInstanceReleased(t *testing.T) {
	log := util.NewLogger("test")
	ctx, cancel := context.WithCancel(t.Context())

	c1, err := Instance(ctx, log, "", "localhost:14197", 0)
	require.NoError(t, err)

	cancel()

	require.Eventually(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		_, ok := instances["uri:localhost:14197"]
		return !ok
	}, time.Second, 10*time.Millisecond)

	c2, err := Instance(t.Context(), log, "", "localhost:14197", 0)
	require.NoError(t, err)
	assert.NotSame(t, c1, c2)

	// late callers must not reopen the port
	_, err = c1.Transact(1, 2, "")
	assert.ErrorIs(t, err, net.ErrClosed)
}

// the connection must survive as long as any charger on the bus is alive
func TestInstanceRefcount(t *testing.T) {
	log := util.NewLogger("test")
	ctx1, cancel1 := context.WithCancel(t.Context())

	c1, err := Instance(ctx1, log, "", "localhost:14198", 0)
	require.NoError(t, err)

	c2, err := Instance(t.Context(), log, "", "localhost:14198", 0)
	require.NoError(t, err)
	require.Same(t, c1, c2)

	cancel1()

	assert.Never(t, func() bool {
		mu.Lock()
		defer mu.Unlock()
		return instances["uri:localhost:14198"] != c2
	}, 100*time.Millisecond, 10*time.Millisecond, "connection released while still in use")
}
