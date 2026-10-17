package modbus

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	gridx "github.com/grid-x/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConcurrentRead(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	srv, _ := mbserver.New(&echoHandler{
		id:             0,
		RequestHandler: new(mbserver.DummyHandler),
	})
	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	var wg sync.WaitGroup

	for id := 1; id <= 10; id++ {
		wg.Go(func() {
			// client
			conn, err := modbus.NewConnection(t.Context(), l.Addr().String(), "", "", 0, modbus.Tcp, uint8(id))
			require.NoError(t, err)

			for range 50 {
				addr := uint16(rand.N(200) + 1)
				qty := uint16(rand.N(32) + 1)

				b, err := conn.ReadInputRegisters(addr, qty)
				require.NoError(t, err)

				if err == nil {
					for u := range qty {
						assert.Equal(t, addr^uint16(id)^u, binary.BigEndian.Uint16(b[2*u:]))
					}
				}

				time.Sleep(rand.N(time.Millisecond))
			}
		})
	}

	wg.Wait()
}

func TestReadCoils(t *testing.T) {
	// downstream server
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	srv, _ := mbserver.New(&echoHandler{
		id:             0,
		RequestHandler: new(mbserver.DummyHandler),
	})
	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	// proxy server
	pl, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer pl.Close()

	downstreamConn, err := modbus.NewConnection(t.Context(), l.Addr().String(), "", "", 0, modbus.Tcp, 1)
	require.NoError(t, err)

	proxy, _ := mbserver.New(&handler{
		log:  util.NewLogger("foo"),
		conn: downstreamConn,
	})
	require.NoError(t, proxy.Start(pl))
	defer func() { _ = proxy.Stop() }()

	// test client
	{
		conn, err := modbus.NewConnection(t.Context(), pl.Addr().String(), "", "", 0, modbus.Tcp, 1)
		require.NoError(t, err)

		{ // read
			b, err := conn.ReadCoils(1, 1)
			require.NoError(t, err)
			assert.Equal(t, []byte{0x01}, b)

			b, err = conn.ReadCoils(1, 2)
			require.NoError(t, err)
			assert.Equal(t, []byte{0x03}, b)

			b, err = conn.ReadCoils(1, 9)
			require.NoError(t, err)
			assert.Equal(t, []byte{0xFF, 0x01}, b)
		}
		{ // write
			b, err := conn.WriteSingleCoil(1, 0xFF00)
			require.NoError(t, err)
			assert.Equal(t, []byte{0xFF, 0x00}, b)

			b, err = conn.WriteMultipleCoils(1, 9, []byte{0xFF, 0x01})
			require.NoError(t, err)
			assert.Equal(t, []byte{0x00, 0x09}, b)
		}
	}
}

func TestUpstreamTimeout(t *testing.T) {
	// downstream server responding slower than the proxy timeout
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer l.Close()

	srv, _ := mbserver.New(&slowHandler{
		delay:          500 * time.Millisecond,
		RequestHandler: new(mbserver.DummyHandler),
	})
	require.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	// proxy server
	pl, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	defer pl.Close()

	downstreamConn, err := modbus.Settings{URI: l.Addr().String(), ID: 1, Timeout: 100 * time.Millisecond}.Connection(t.Context())
	require.NoError(t, err)

	proxy, _ := mbserver.New(&handler{
		log:  util.NewLogger("foo"),
		conn: downstreamConn,
	})
	require.NoError(t, proxy.Start(pl))
	defer func() { _ = proxy.Stop() }()

	// test client
	conn, err := modbus.NewConnection(t.Context(), pl.Addr().String(), "", "", 0, modbus.Tcp, 1)
	require.NoError(t, err)
	conn.Timeout(5 * time.Second)

	_, err = conn.WriteMultipleRegisters(1, 1, []byte{0, 1})
	me, ok := errors.AsType[*gridx.Error](err)
	require.True(t, ok, "expected modbus exception, got %v", err)
	assert.Equal(t, uint8(gridx.ExceptionCodeGatewayTargetDeviceFailedToRespond), me.ExceptionCode)
}

type slowHandler struct {
	delay time.Duration
	mbserver.RequestHandler
}

func (h *slowHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	time.Sleep(h.delay)
	return req.Args, nil
}

type echoHandler struct {
	id int
	mbserver.RequestHandler
}

func (h *echoHandler) HandleInputRegisters(req *mbserver.InputRegistersRequest) (res []uint16, err error) {
	for u := uint16(0); u < req.Quantity; u++ {
		res = append(res, req.Addr^uint16(req.UnitId)^u)
	}

	return res, err
}

func (h *echoHandler) HandleCoils(req *mbserver.CoilsRequest) (res []bool, err error) {
	if req.IsWrite {
		return nil, nil
	}

	for u := uint16(0); u < req.Quantity; u++ {
		res = append(res, true)
	}

	return res, err
}

func TestRawFunctionCodes(t *testing.T) {
	// Huawei start file upload: sub function 0x05, data length 1, file type 0x45
	req := []byte{0x05, 0x01, 0x45}

	t.Run("forwarded", func(t *testing.T) {
		addr, _ := startRawProxy(t, &rawEchoHandler{RequestHandler: new(mbserver.DummyHandler)}, ReadOnlyFalse)

		b, err := rawClient(t, addr, 16).SendRaw(0x41, req)
		require.NoError(t, err)
		// downstream echoes unit id and request data
		assert.Equal(t, append([]byte{16}, req...), b)
	})

	t.Run("vendor exception passed through", func(t *testing.T) {
		// Huawei permission denied
		addr, _ := startRawProxy(t, &exceptionHandler{RequestHandler: new(mbserver.DummyHandler), code: 0x80}, ReadOnlyFalse)

		_, err := rawClient(t, addr, 16).SendRaw(0x41, req)
		assertException(t, err, 0xc1, 0x80)
	})

	for _, mode := range []ReadOnlyMode{ReadOnlyTrue, ReadOnlyDeny} {
		t.Run("rejected if read-only "+mode.String(), func(t *testing.T) {
			downstream := &rawEchoHandler{RequestHandler: new(mbserver.DummyHandler)}
			addr, _ := startRawProxy(t, downstream, mode)

			_, err := rawClient(t, addr, 16).SendRaw(0x41, req)
			assertException(t, err, 0xc1, gridx.ExceptionCodeIllegalFunction)
			assert.Zero(t, downstream.calls.Load())
		})
	}

	t.Run("rejected for non-tcp downstream", func(t *testing.T) {
		// rtu framing over tcp
		conn, err := modbus.NewConnection(t.Context(), "localhost:1", "", "", 0, modbus.Rtu, 1)
		require.NoError(t, err)

		_, err = rawClient(t, startProxy(t, conn, ReadOnlyFalse), 16).SendRaw(0x41, req)
		assertException(t, err, 0xc1, gridx.ExceptionCodeIllegalFunction)
	})

	t.Run("upstream timeout reported as gateway target failure", func(t *testing.T) {
		// downstream server responding slower than the proxy timeout
		l, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		t.Cleanup(func() { l.Close() })

		srv, err := mbserver.New(&slowRawHandler{
			delay:          500 * time.Millisecond,
			RequestHandler: new(mbserver.DummyHandler),
		})
		require.NoError(t, err)
		require.NoError(t, srv.Start(l))
		t.Cleanup(func() { _ = srv.Stop() })

		downstreamConn, err := modbus.Settings{URI: l.Addr().String(), ID: 1, Timeout: 100 * time.Millisecond}.Connection(t.Context())
		require.NoError(t, err)

		conn := rawClient(t, startProxy(t, downstreamConn, ReadOnlyFalse), 16)
		conn.Timeout(5 * time.Second)

		_, err = conn.SendRaw(0x41, req)
		assertException(t, err, 0xc1, gridx.ExceptionCodeGatewayTargetDeviceFailedToRespond)
	})
}

func TestVendorExceptionPassedThrough(t *testing.T) {
	// Huawei permission denied for a standard function code
	addr, _ := startRawProxy(t, &exceptionHandler{RequestHandler: new(mbserver.DummyHandler), code: 0x80}, ReadOnlyFalse)

	_, err := rawClient(t, addr, 16).ReadInputRegisters(0, 1)
	assertException(t, err, 0x84, 0x80)
}

// assertException asserts that err is a modbus exception response
func assertException(t *testing.T, err error, functionCode, exceptionCode byte) {
	t.Helper()

	if me, ok := errors.AsType[*gridx.Error](err); assert.True(t, ok, "expected modbus exception, got: %v", err) {
		assert.Equal(t, functionCode, me.FunctionCode)
		assert.Equal(t, exceptionCode, me.ExceptionCode)
	}
}

func TestRawConcurrentClients(t *testing.T) {
	// downstream answers raw requests and input register reads
	addr, downstream := startRawProxy(t, &rawEchoHandler{
		RequestHandler: &echoHandler{RequestHandler: new(mbserver.DummyHandler)},
	}, ReadOnlyFalse)

	// independent proxy clients, each with its own tcp connection
	clients := make([]net.Conn, 5)
	for i := range clients {
		conn, err := net.Dial("tcp", addr)
		require.NoError(t, err)
		t.Cleanup(func() { conn.Close() })
		clients[i] = conn
	}

	start := make(chan struct{})

	var wg sync.WaitGroup

	for i, conn := range clients {
		// unit ids differ from the downstream connection's id 1
		id := byte(i + 2)

		wg.Go(func() {
			<-start

			for n := range 50 {
				if n%2 == 0 {
					// Huawei upload file frame: sub function 0x06, data length 3, file type, frame number
					req := []byte{0x41, 0x06, 0x03, 0x45, 0x00, byte(n)}

					res, err := rawRoundTrip(conn, uint16(n), id, req)
					if !assert.NoError(t, err) {
						return
					}

					// downstream echoes unit id and request data
					assert.Equal(t, append([]byte{0x41, id}, req[1:]...), res)

					continue
				}

				reg := uint16(rand.N(200) + 1)
				qty := uint16(rand.N(32) + 1)
				req := binary.BigEndian.AppendUint16(binary.BigEndian.AppendUint16([]byte{0x04}, reg), qty)

				res, err := rawRoundTrip(conn, uint16(n), id, req)
				if !assert.NoError(t, err) || !assert.Len(t, res, 2+2*int(qty)) {
					return
				}

				assert.Equal(t, byte(0x04), res[0])
				assert.Equal(t, byte(2*qty), res[1])

				for u := range qty {
					assert.Equal(t, reg^uint16(id)^u, binary.BigEndian.Uint16(res[2+2*u:]))
				}
			}
		})
	}

	// evcc polling on the shared downstream connection
	wg.Go(func() {
		<-start

		for range 50 {
			reg := uint16(rand.N(200) + 1)
			qty := uint16(rand.N(32) + 1)

			b, err := downstream.ReadInputRegisters(reg, qty)
			if !assert.NoError(t, err) || !assert.Len(t, b, 2*int(qty)) {
				return
			}

			for u := range qty {
				assert.Equal(t, reg^1^u, binary.BigEndian.Uint16(b[2*u:]))
			}
		}
	})

	close(start)
	wg.Wait()
}

func rawClient(t *testing.T, addr string, id uint8) *modbus.Connection {
	t.Helper()

	conn, err := modbus.NewConnection(t.Context(), addr, "", "", 0, modbus.Tcp, id)
	require.NoError(t, err)

	return conn
}

// rawRoundTrip sends a request PDU over a plain Modbus TCP connection and returns the response PDU
func rawRoundTrip(conn net.Conn, txn uint16, id uint8, req []byte) ([]byte, error) {
	if err := conn.SetDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return nil, err
	}

	// MBAP header: transaction id, protocol id, length, unit id
	frame := binary.BigEndian.AppendUint16(nil, txn)
	frame = binary.BigEndian.AppendUint16(frame, 0)
	frame = binary.BigEndian.AppendUint16(frame, uint16(len(req)+1))
	frame = append(append(frame, id), req...)

	if _, err := conn.Write(frame); err != nil {
		return nil, err
	}

	header := make([]byte, 7)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}

	// length covers unit id and PDU (at most 253 bytes)
	length := binary.BigEndian.Uint16(header[4:])
	if binary.BigEndian.Uint16(header) != txn || binary.BigEndian.Uint16(header[2:]) != 0 || header[6] != id ||
		length < 2 || length > 254 {
		return nil, fmt.Errorf("unexpected response header % x", header)
	}

	res := make([]byte, int(length)-1)
	_, err := io.ReadFull(conn, res)

	return res, err
}

// startRawProxy starts a downstream server using the given handler and a proxy in front of it.
// It returns the proxy address and the proxy's downstream connection.
func startRawProxy(t *testing.T, downstream mbserver.RequestHandler, readOnly ReadOnlyMode) (string, *modbus.Connection) {
	t.Helper()

	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })

	srv, err := mbserver.New(downstream)
	require.NoError(t, err)
	require.NoError(t, srv.Start(l))
	t.Cleanup(func() { _ = srv.Stop() })

	conn, err := modbus.NewConnection(t.Context(), l.Addr().String(), "", "", 0, modbus.Tcp, 1)
	require.NoError(t, err)

	return startProxy(t, conn, readOnly), conn
}

// startProxy starts a proxy in front of the given downstream connection and returns its address
func startProxy(t *testing.T, conn *modbus.Connection, readOnly ReadOnlyMode) string {
	t.Helper()

	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })

	proxy, err := mbserver.New(&handler{
		log:      util.NewLogger("foo"),
		readOnly: readOnly,
		conn:     conn,
	})
	require.NoError(t, err)
	require.NoError(t, proxy.Start(l))
	t.Cleanup(func() { _ = proxy.Stop() })

	return l.Addr().String()
}

type rawEchoHandler struct {
	mbserver.RequestHandler
	calls atomic.Int32
}

func (h *rawEchoHandler) HandleRaw(req *mbserver.RawRequest) ([]byte, error) {
	h.calls.Add(1)
	return append([]byte{req.UnitId}, req.Payload...), nil
}

// exceptionHandler answers raw requests and input register reads with an exception
type exceptionHandler struct {
	mbserver.RequestHandler
	code byte
}

func (h *exceptionHandler) HandleRaw(*mbserver.RawRequest) ([]byte, error) {
	return nil, mbserver.ExceptionError(h.code)
}

func (h *exceptionHandler) HandleInputRegisters(*mbserver.InputRegistersRequest) ([]uint16, error) {
	return nil, mbserver.ExceptionError(h.code)
}

type slowRawHandler struct {
	delay time.Duration
	mbserver.RequestHandler
}

func (h *slowRawHandler) HandleRaw(req *mbserver.RawRequest) ([]byte, error) {
	time.Sleep(h.delay)
	return req.Payload, nil
}
