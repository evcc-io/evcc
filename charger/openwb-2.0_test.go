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
	mu           sync.Mutex
	failReset    bool
	blankRFID    bool
	failRFIDRead bool
	rfidReads    int
	rfidResets   int
	resetWrites  [][2]uint16
}

func (h *openWB20Handler) HandleInputRegisters(req *mbserver.InputRegistersRequest) ([]uint16, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

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
	h.mu.Lock()
	defer h.mu.Unlock()

	if req.IsWrite && req.Addr == openwbRegResetRfid && h.failReset {
		return nil, mbserver.ErrIllegalDataValue
	}
	if req.IsWrite && req.Addr == openwbRegResetRfid {
		h.rfidResets++
		h.resetWrites = append(h.resetWrites, [2]uint16{req.Addr, req.Args[0]})
	}
	return req.Args, nil
}

func (h *openWB20Handler) setState(failReset, blankRFID, failRFIDRead bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	h.failReset = failReset
	h.blankRFID = blankRFID
	h.failRFIDRead = failRFIDRead
}

func (h *openWB20Handler) counts() (int, int, [][2]uint16) {
	h.mu.Lock()
	defer h.mu.Unlock()

	return h.rfidReads, h.rfidResets, append([][2]uint16(nil), h.resetWrites...)
}

func TestOpenWB20PendingRfidReset(t *testing.T) {
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)

	handler := &openWB20Handler{RequestHandler: new(mbserver.DummyHandler), failReset: true}
	server, err := mbserver.New(handler)
	require.NoError(t, err)
	require.NoError(t, server.Start(listener))
	t.Cleanup(func() {
		require.NoError(t, server.Stop())
	})

	conn, err := modbus.NewConnection(context.Background(), listener.Addr().String(), "", "", 0, modbus.Tcp, 1)
	require.NoError(t, err)

	wb := &OpenWB20{
		log:             util.NewLogger("openwb-2.0"),
		conn:            conn,
		laststatus:      api.StatusB,
		identifyEnabled: true,
	}

	// A failed reset on disconnect must stay pending and prevent RFID reads.
	status, err := wb.Status()
	require.NoError(t, err)
	require.Equal(t, api.StatusA, status)
	require.Equal(t, api.StatusA, wb.laststatus)
	require.True(t, wb.rfidResetPending)
	reads, _, _ := handler.counts()
	require.Zero(t, reads)

	// Identification retries the reset before attempting to read the tag.
	identity, err := wb.identify()
	require.Error(t, err)
	require.ErrorContains(t, err, "reset RFID")
	require.Empty(t, identity)
	reads, _, _ = handler.counts()
	require.Zero(t, reads)

	// Once the reset succeeds, identification can read the tag and clear the pending flag.
	handler.setState(false, false, false)
	identity, err = wb.identify()
	require.NoError(t, err)
	require.Equal(t, []string{"tag"}, identity)
	require.False(t, wb.rfidResetPending)
	reads, _, writes := handler.counts()
	require.Equal(t, 1, reads)
	require.Equal(t, [][2]uint16{{openwbRegResetRfid, 1}}, writes)

	// A missing tag clears the remembered identity and its age.
	handler.setState(false, true, false)
	wb.laststatus = api.StatusA
	wb.lastidentify = []string{"tag"}
	wb.lastidtime = time.Now().Add(-3 * time.Minute)

	identity, err = wb.identify()
	require.NoError(t, err)
	require.Nil(t, identity)

	wb.handleRfidReset()
	require.False(t, wb.rfidResetPending)
	require.Nil(t, wb.lastidentify)
	require.True(t, wb.lastidtime.IsZero())
	_, resets, _ := handler.counts()
	require.Equal(t, 1, resets)

	// RFID read failures are reported rather than returned as an empty identity.
	handler.setState(false, false, true)
	_, err = wb.identify()
	require.ErrorContains(t, err, "read RFID")
}
