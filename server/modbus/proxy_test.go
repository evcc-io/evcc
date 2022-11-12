package modbus

import (
	"encoding/binary"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/assert"
)

func TestProxyRead(t *testing.T) {
	l, err := net.Listen("tcp", ":0")
	assert.NoError(t, err)
	defer l.Close()

	t.Log(l.Addr().String())

	conn, err := modbus.NewConnection(l.Addr().String(), "", "", 0, modbus.Tcp, 1)
	assert.NoError(t, err)

	h := &echoHandler{
		id:             0,
		RequestHandler: new(mbserver.DummyHandler),
		conn:           conn,
	}

	srv, _ := mbserver.New(h)
	assert.NoError(t, srv.Start(l))
	defer func() { _ = srv.Stop() }()

	var wg sync.WaitGroup

	for i := 1; i <= 10; i++ {
		wg.Add(1)

		go func(id int) {
			for i := 0; i < 50; i++ {
				addr := uint16(rand.Int31n(200) + 1)

				b, err := conn.ReadInputRegistersWithSlave(uint8(id), addr, 1)
				assert.NoError(t, err)

				if err == nil {
					assert.Equal(t, addr^uint16(id), binary.BigEndian.Uint16(b))
				}

				time.Sleep(time.Duration(rand.Int31n(1000)) * time.Microsecond)
			}

			wg.Done()
		}(i)
	}

	wg.Wait()
}

type echoHandler struct {
	id int
	mbserver.RequestHandler
	conn *modbus.Connection
}

func (h *echoHandler) HandleInputRegisters(req *mbserver.InputRegistersRequest) (res []uint16, err error) {
	return []uint16{req.Addr ^ uint16(req.UnitId)}, err
}
