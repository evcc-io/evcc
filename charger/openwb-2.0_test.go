package charger

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/require"
)

type openWB20Handler struct {
	mbserver.RequestHandler
	mu        sync.Mutex
	plugged   uint16
	failReset bool
	resets    int
}

func (h *openWB20Handler) HandleInputRegisters(req *mbserver.InputRegistersRequest) ([]uint16, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	res := make([]uint16, req.Quantity)
	if req.Addr == openwbRegPlugged {
		res[0] = h.plugged
	}
	return res, nil
}

func (h *openWB20Handler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if req.IsWrite && req.Addr == openwbRegResetRfid {
		if h.failReset {
			return nil, mbserver.ErrIllegalDataValue
		}
		h.resets++
	}
	return req.Args, nil
}

func (h *openWB20Handler) set(plugged uint16, failReset bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.plugged, h.failReset = plugged, failReset
}

func (h *openWB20Handler) count() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.resets
}

func TestOpenWB20RfidReset(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	h := &openWB20Handler{RequestHandler: new(mbserver.DummyHandler), plugged: 1}
	srv, err := mbserver.New(h)
	require.NoError(t, err)
	require.NoError(t, srv.Start(l))
	t.Cleanup(func() { require.NoError(t, srv.Stop()) })

	conn, err := modbus.NewConnection(context.Background(), l.Addr().String(), "", "", 0, modbus.Tcp, 1)
	require.NoError(t, err)

	wb := &OpenWB20{conn: conn, rfid: true}

	status, err := wb.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusB, status)
	require.Zero(t, h.count())

	// failed reset on disconnect is retried
	h.set(0, true)
	_, err = wb.Status()
	require.ErrorContains(t, err, "reset rfid")

	h.set(0, false)
	status, err = wb.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusA, status)
	require.Equal(t, 1, h.count())

	// no further resets while disconnected
	_, err = wb.Status()
	require.NoError(t, err)
	require.Equal(t, 1, h.count())
}
