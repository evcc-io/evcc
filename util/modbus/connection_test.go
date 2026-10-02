package modbus

import (
	"io"
	"net"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestReadReissuesAfterPeerClose ensures a read is repeated once on a fresh
// connection when the device dropped the idle one (Sungrow SH10RT after ~16s).
func TestReadReissuesAfterPeerClose(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	var accepted atomic.Int32

	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}

			// peer drops the first connection before answering
			if accepted.Add(1) == 1 {
				c.Close()
				continue
			}

			go func() {
				defer c.Close()

				req := make([]byte, 12)
				if _, err := io.ReadFull(c, req); err != nil {
					return
				}

				// MBAP header, unit id, fc04, byte count, one register
				res := append(append([]byte{}, req[:4]...), 0, 5, req[6], 4, 2, 0x12, 0x34)
				_, _ = c.Write(res)
			}()
		}
	}()

	conn, err := NewConnection(t.Context(), l.Addr().String(), "", "", 0, Tcp, 1)
	require.NoError(t, err)

	b, err := conn.ReadInputRegisters(1, 1)
	require.NoError(t, err)
	assert.Equal(t, []byte{0x12, 0x34}, b)
	assert.EqualValues(t, 2, accepted.Load())
}
