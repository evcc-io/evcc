package modbus

import (
	"encoding/binary"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/assert"
)

func TestConcurrentRead(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)
	defer l.Close()

	srv, _ := mbserver.New(&echoHandler{
		id:             0,
		RequestHandler: new(mbserver.DummyHandler),
	})
	assert.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	// client
	conn, err := modbus.NewConnection(l.Addr().String(), "", "", 0, modbus.Tcp, 1)
	assert.NoError(t, err)

	var wg sync.WaitGroup

	for i := 1; i <= 10; i++ {
		wg.Add(1)

		go func(id int) {
			for i := 0; i < 50; i++ {
				addr := uint16(rand.Int31n(200) + 1)
				qty := uint16(rand.Int31n(32) + 1)

				b, err := conn.ReadInputRegistersWithSlave(uint8(id), addr, qty)
				assert.NoError(t, err)

				if err == nil {
					for u := uint16(0); u < qty; u++ {
						assert.Equal(t, addr^uint16(id)^u, binary.BigEndian.Uint16(b[2*u:]))
					}
				}

				time.Sleep(time.Duration(rand.Int31n(1000)) * time.Microsecond)
			}

			wg.Done()
		}(i)
	}

	wg.Wait()
}

func TestReadCoils(t *testing.T) {
	// downstream server
	l, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)
	defer l.Close()

	srv, _ := mbserver.New(&echoHandler{
		id:             0,
		RequestHandler: new(mbserver.DummyHandler),
	})
	assert.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	// proxy server
	pl, err := net.Listen("tcp", "localhost:0")
	assert.NoError(t, err)
	defer pl.Close()

	downstreamConn, err := modbus.NewConnection(l.Addr().String(), "", "", 0, modbus.Tcp, 1)
	assert.NoError(t, err)

	proxy, _ := mbserver.New(&handler{
		log:            util.NewLogger("foo"),
		RequestHandler: new(mbserver.DummyHandler),
		conn:           downstreamConn,
	})
	assert.NoError(t, proxy.Start(pl))
	defer func() { _ = proxy.Stop() }()

	// test client
	{
		conn, err := modbus.NewConnection(pl.Addr().String(), "", "", 0, modbus.Tcp, 1)
		assert.NoError(t, err)

		{ // read
			b, err := conn.ReadCoilsWithSlave(1, 1, 1)
			assert.NoError(t, err)
			assert.Equal(t, []byte{0x01}, b)

			b, err = conn.ReadCoilsWithSlave(1, 1, 2)
			assert.NoError(t, err)
			assert.Equal(t, []byte{0x03}, b)

			b, err = conn.ReadCoilsWithSlave(1, 1, 9)
			assert.NoError(t, err)
			assert.Equal(t, []byte{0xFF, 0x01}, b)
		}
		{ // write
			b, err := conn.WriteSingleCoilWithSlave(1, 1, 0xFF00)
			assert.NoError(t, err)
			assert.Equal(t, []byte{0xFF, 0x00}, b)

			b, err = conn.WriteMultipleCoilsWithSlave(1, 1, 9, []byte{0xFF, 0x01})
			assert.NoError(t, err)
			assert.Equal(t, []byte{0x00, 0x09}, b)
		}
	}
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
