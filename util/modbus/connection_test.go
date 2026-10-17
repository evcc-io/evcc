package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"net"
	"sync/atomic"
	"testing"

	gridx "github.com/grid-x/modbus"
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

// fakeDevice counts connections and requests of a fake Modbus TCP device
type fakeDevice struct {
	accepted atomic.Int32
	requests atomic.Int32
}

// startFakeDevice starts a fake Modbus TCP device answering each request PDU with the PDU
// returned by respond. A nil response closes the connection without answering.
func startFakeDevice(t *testing.T, respond func(n int32, req []byte) []byte) (string, *fakeDevice) {
	t.Helper()

	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })

	dev := new(fakeDevice)

	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}

			dev.accepted.Add(1)

			go func() {
				defer c.Close()

				for {
					header := make([]byte, 7)
					if _, err := io.ReadFull(c, header); err != nil {
						return
					}

					// length covers unit id and PDU
					length := binary.BigEndian.Uint16(header[4:])
					if length < 2 || length > 2+maxRawDataLength {
						return
					}

					req := make([]byte, int(length)-1)
					if _, err := io.ReadFull(c, req); err != nil {
						return
					}

					res := respond(dev.requests.Add(1), req)
					if res == nil {
						return
					}

					// echo transaction id, protocol id and unit id
					frame := binary.BigEndian.AppendUint16(append([]byte{}, header[:4]...), uint16(len(res)+1))
					frame = append(append(frame, header[6]), res...)
					if _, err := c.Write(frame); err != nil {
						return
					}
				}
			}()
		}
	}()

	return l.Addr().String(), dev
}

// Huawei start file upload: sub function 0x05, data length 1, file type 0x45
var rawRequest = []byte{0x05, 0x01, 0x45}

func TestSendRawNotRetried(t *testing.T) {
	// device receives the request and closes the connection without answering
	addr, dev := startFakeDevice(t, func(int32, []byte) []byte { return nil })

	conn, err := NewConnection(t.Context(), addr, "", "", 0, Tcp, 1)
	require.NoError(t, err)

	_, err = conn.SendRaw(0x41, rawRequest)
	require.Error(t, err)
	assert.EqualValues(t, 1, dev.requests.Load())
}

func TestSendRawExceptionKeepsConnection(t *testing.T) {
	addr, dev := startFakeDevice(t, func(n int32, req []byte) []byte {
		if n == 1 {
			// Huawei permission denied
			return []byte{0x80 | req[0], 0x80}
		}
		return req
	})

	conn, err := NewConnection(t.Context(), addr, "", "", 0, Tcp, 1)
	require.NoError(t, err)

	_, err = conn.SendRaw(0x41, rawRequest)
	me, ok := errors.AsType[*gridx.Error](err)
	require.True(t, ok, "expected modbus exception, got: %v", err)
	assert.Equal(t, byte(0xc1), me.FunctionCode)
	assert.Equal(t, byte(0x80), me.ExceptionCode)

	b, err := conn.SendRaw(0x41, rawRequest)
	require.NoError(t, err)
	assert.Equal(t, rawRequest, b)

	assert.EqualValues(t, 1, dev.accepted.Load())
}

func TestSendRawInvalidResponse(t *testing.T) {
	for _, res := range [][]byte{
		{0x03, 0x00},       // other function code
		{0xc1},             // exception without code
		{0xc1, 0x80, 0x00}, // exception with trailing data
	} {
		t.Run(fmt.Sprintf("% x", res), func(t *testing.T) {
			addr, _ := startFakeDevice(t, func(int32, []byte) []byte { return res })

			conn, err := NewConnection(t.Context(), addr, "", "", 0, Tcp, 1)
			require.NoError(t, err)

			_, err = conn.SendRaw(0x41, rawRequest)
			require.Error(t, err)

			_, ok := errors.AsType[*gridx.Error](err)
			assert.False(t, ok, "invalid response reported as exception")
		})
	}
}

func TestSendRawUnsupported(t *testing.T) {
	// rtu framing over tcp
	conn, err := NewConnection(t.Context(), "localhost:1", "", "", 0, Rtu, 1)
	require.NoError(t, err)

	_, err = conn.SendRaw(0x41, rawRequest)
	assert.ErrorIs(t, err, ErrRawUnsupported)
}

func TestSendRawTooLong(t *testing.T) {
	addr, dev := startFakeDevice(t, func(_ int32, req []byte) []byte { return req })

	conn, err := NewConnection(t.Context(), addr, "", "", 0, Tcp, 1)
	require.NoError(t, err)

	_, err = conn.SendRaw(0x41, make([]byte, maxRawDataLength+1))
	assert.Error(t, err)
	assert.Zero(t, dev.requests.Load())
}
