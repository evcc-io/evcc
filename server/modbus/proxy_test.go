package modbus

import (
	"encoding/binary"
	"errors"
	"math/rand/v2"
	"net"
	"sync"
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
