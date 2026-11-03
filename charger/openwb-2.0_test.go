package charger

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/require"
)

type openWB20Handler struct {
	mbserver.RequestHandler
	failReset    bool
	blankRFID    bool
	failRFIDRead bool
	rfidReads    int
	rfidResets   int
}

func (h *openWB20Handler) HandleInputRegisters(req *mbserver.InputRegistersRequest) ([]uint16, error) {
	res := make([]uint16, req.Quantity)
	if h.failRFIDRead && req.Addr <= openwbRegRfid && openwbRegRfid < req.Addr+req.Quantity {
		return nil, mbserver.ErrIllegalDataAddress
	}
	if h.blankRFID {
		return res, nil
	}
	for i := range res {
		res[i] = ' '<<8 | ' '
	}
	if req.Addr <= openwbRegRfid && openwbRegRfid < req.Addr+req.Quantity {
		h.rfidReads++
		res[openwbRegRfid-req.Addr] = uint16('t')<<8 | uint16('a')
		res[openwbRegRfid-req.Addr+1] = uint16('g')<<8 | ' '
	}
	return res, nil
}

func (h *openWB20Handler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	if req.IsWrite && req.Addr == openwbRegResetRfid && h.failReset {
		return nil, mbserver.ErrIllegalDataValue
	}
	if req.IsWrite && req.Addr == openwbRegResetRfid {
		h.rfidResets++
	}
	return req.Args, nil
}

func TestOpenWB20PendingRfidReset(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	handler := &openWB20Handler{RequestHandler: new(mbserver.DummyHandler), failReset: true}
	server, err := mbserver.New(handler)
	require.NoError(t, err)
	require.NoError(t, server.Start(listener))

	conn, err := modbus.NewConnection(context.Background(), listener.Addr().String(), "", "", 0, modbus.Tcp, 1)
	require.NoError(t, err)

	wb := &OpenWB20{log: util.NewLogger("openwb-2.0"), conn: conn, laststatus: api.StatusB}
	status, err := wb.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusA, status)
	require.Equal(t, api.StatusA, wb.laststatus)
	require.True(t, wb.rfidResetPending)
	require.Zero(t, handler.rfidReads)

	identity, err := wb.identify()
	require.Error(t, err)
	require.ErrorContains(t, err, "reset RFID")
	require.Empty(t, identity)
	require.Zero(t, handler.rfidReads)

	handler.failReset = false
	identity, err = wb.identify()
	require.NoError(t, err)
	require.Equal(t, []string{"tag"}, identity)
	require.False(t, wb.rfidResetPending)
	require.Equal(t, 1, handler.rfidReads)

	handler.blankRFID = true
	wb.laststatus = api.StatusA
	wb.lastidentify = []string{"tag"}
	wb.lastidtime = time.Now().Add(-3 * time.Minute)

	identity, err = wb.identify()
	require.NoError(t, err)
	require.Nil(t, identity)

	require.NoError(t, wb.handleRfidReset())
	require.Nil(t, wb.lastidentify)
	require.Equal(t, time.Unix(0, 0), wb.lastidtime)
	require.Equal(t, 1, handler.rfidResets)

	handler.failReset = true
	wb.rfidResetPending = true
	var wg sync.WaitGroup
	for range 3 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				_, _ = wb.Status()
				_ = wb.handleRfidReset()
				_, _ = wb.identify()
			}
		}()
	}
	wg.Wait()

	handler.failReset = false
	handler.failRFIDRead = true
	_, err = wb.identify()
	require.ErrorContains(t, err, "read RFID")
}
