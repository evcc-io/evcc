package charger

import (
	"net"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type weishauptHandler struct {
	mbserver.DummyHandler
	power atomic.Uint32
	fail  atomic.Bool
}

func (h *weishauptHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	if req.UnitId != 1 || req.Addr != 40002 || req.Quantity != 1 {
		return nil, mbserver.ErrIllegalDataAddress
	}
	if h.fail.Load() {
		return nil, mbserver.ErrServerDeviceFailure
	}
	if req.IsWrite {
		if req.WriteFuncCode != 6 {
			return nil, mbserver.ErrIllegalFunction
		}
		h.power.Store(uint32(req.Args[0]))
	}
	return []uint16{uint16(h.power.Load())}, nil
}

// Keep one server alive because mbserver.Stop races its accept goroutine.
var (
	weishauptOnce sync.Once
	weishauptURI  string
	weishauptH    weishauptHandler
)

func weishauptTestCharger(t *testing.T) *Weishaupt {
	t.Helper()
	weishauptOnce.Do(func() {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		require.NoError(t, err)
		server, err := mbserver.New(&weishauptH)
		require.NoError(t, err)
		require.NoError(t, server.Start(listener))
		weishauptURI = listener.Addr().String()
	})
	weishauptH.power.Store(0)
	weishauptH.fail.Store(false)

	charger, err := NewWeishaupt(t.Context(), modbus.Settings{URI: weishauptURI, ID: 1}, "warmwater")
	require.NoError(t, err)
	wb := charger.(*Weishaupt)
	t.Cleanup(wb.conn.Close)
	return wb
}

func TestWeishauptEnable(t *testing.T) {
	for _, tc := range []struct {
		name     string
		currents []float64
		enable   bool
		power    uint32
	}{
		{"initial enable", nil, true, 1},
		{"initial disable", nil, false, 0},
		{"resume", []float64{10.5}, true, 2415},
		{"resume after zero", []float64{10.5, 0}, true, 2415},
		{"enable after zero", []float64{0}, true, 1},
		{"disable", []float64{10.5}, false, 0},
		{"capped power", []float64{1000}, true, 65535},
	} {
		t.Run(tc.name, func(t *testing.T) {
			wb := weishauptTestCharger(t)
			for _, current := range tc.currents {
				require.NoError(t, wb.MaxCurrentMillis(current))
			}
			require.NoError(t, wb.Enable(false))

			require.NoError(t, wb.Enable(tc.enable))

			assert.Equal(t, tc.power, weishauptH.power.Load())
		})
	}
}

func TestWeishauptEnabled(t *testing.T) {
	wb := weishauptTestCharger(t)
	for _, power := range []uint32{0, 1, 65535} {
		weishauptH.power.Store(power)
		enabled, err := wb.Enabled()
		require.NoError(t, err)
		assert.Equal(t, power > 0, enabled)
	}

	weishauptH.fail.Store(true)
	_, err := wb.Enabled()
	assert.Error(t, err)
}

func TestWeishauptPowerWrite(t *testing.T) {
	wb := weishauptTestCharger(t)
	require.NoError(t, wb.MaxCurrent(10))
	assert.Equal(t, uint32(2300), weishauptH.power.Load())
	require.NoError(t, wb.MaxCurrent(0))
	assert.Zero(t, weishauptH.power.Load())

	weishauptH.fail.Store(true)
	assert.Error(t, wb.MaxCurrent(20))
	assert.Error(t, wb.Enable(true))
	assert.Error(t, wb.Enable(false))
	weishauptH.fail.Store(false)

	require.NoError(t, wb.Enable(true))
	assert.Equal(t, uint32(2300), weishauptH.power.Load())
}
