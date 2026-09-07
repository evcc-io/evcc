package charger

import (
	"context"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type weishauptHandler struct {
	mbserver.DummyHandler
	power      atomic.Uint32
	reads      atomic.Uint32
	writes     atomic.Uint32
	fail       atomic.Bool
	failWrites atomic.Bool
}

func (h *weishauptHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	if req.UnitId != 1 || req.Addr != 40002 || req.Quantity != 1 {
		return nil, mbserver.ErrIllegalDataAddress
	}
	if !req.IsWrite {
		h.reads.Add(1)
	}
	if h.fail.Load() {
		return nil, mbserver.ErrServerDeviceFailure
	}
	if req.IsWrite {
		if req.WriteFuncCode != 6 {
			return nil, mbserver.ErrIllegalFunction
		}
		if h.failWrites.Load() {
			return nil, mbserver.ErrServerDeviceFailure
		}
		h.power.Store(uint32(req.Args[0]))
		h.writes.Add(1)
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
	weishauptH.reads.Store(0)
	weishauptH.writes.Store(0)
	weishauptH.fail.Store(false)
	weishauptH.failWrites.Store(false)

	charger, err := NewWeishauptFromConfig(t.Context(), map[string]any{"uri": weishauptURI})
	require.NoError(t, err)
	wb := charger.(*Weishaupt)
	t.Cleanup(wb.conn.Close)
	return wb
}

func TestWeishauptConfig(t *testing.T) {
	for _, tc := range []struct {
		name     string
		config   map[string]any
		icon     string
		features []api.Feature
	}{
		{"defaults", map[string]any{}, "heatpump", []api.Feature{api.Continuous, api.Heating, api.IntegratedDevice}},
		{"override", map[string]any{"icon": "heater", "features": []api.Feature{api.Heating}}, "heater", []api.Feature{api.Heating}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tc.config["uri"] = "127.0.0.1:502"
			charger, err := NewWeishauptFromConfig(t.Context(), tc.config)
			require.NoError(t, err)
			wb := charger.(*Weishaupt)
			t.Cleanup(wb.conn.Close)

			assert.Equal(t, tc.icon, wb.Icon())
			assert.Equal(t, tc.features, wb.Features())
		})
	}
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

func TestWeishauptStatus(t *testing.T) {
	wb := weishauptTestCharger(t)
	for _, tc := range []struct {
		power  uint32
		status api.ChargeStatus
	}{
		{0, api.StatusB},
		{100, api.StatusB},
		{101, api.StatusC},
		{65535, api.StatusC},
	} {
		weishauptH.power.Store(tc.power)
		status, err := wb.Status()
		require.NoError(t, err)
		assert.Equal(t, tc.status, status)
	}

	weishauptH.fail.Store(true)
	status, err := wb.Status()
	assert.Error(t, err)
	assert.Equal(t, api.StatusNone, status)
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

func TestWeishauptHeartbeat(t *testing.T) {
	for _, name := range []string{"positive", "initial enable", "changed setpoint", "disabled", "zero current", "external disable", "read failure", "write failure"} {
		t.Run(name, func(t *testing.T) {
			wb := weishauptTestCharger(t)
			want := uint32(2300)
			if name == "initial enable" {
				require.NoError(t, wb.Enable(true))
				want = 1
			} else {
				require.NoError(t, wb.MaxCurrent(10))
			}
			switch name {
			case "changed setpoint":
				weishauptH.power.Store(4600)
			case "disabled":
				require.NoError(t, wb.Enable(false))
				want = 0
			case "zero current":
				require.NoError(t, wb.MaxCurrent(0))
				want = 0
			case "external disable":
				weishauptH.power.Store(0)
				want = 0
			case "read failure":
				weishauptH.fail.Store(true)
			case "write failure":
				weishauptH.failWrites.Store(true)
			}
			weishauptH.writes.Store(0)

			ctx, cancel := context.WithCancel(t.Context())
			done := make(chan struct{})
			go func() {
				defer close(done)
				wb.heartbeat(ctx, 10*time.Millisecond)
			}()
			stop := func() {
				cancel()
				select {
				case <-done:
				case <-time.After(time.Second):
					t.Fatal("heartbeat did not stop")
				}
			}
			t.Cleanup(stop)

			require.Eventually(t, func() bool { return weishauptH.reads.Load() >= 2 }, time.Second, time.Millisecond)
			if name == "read failure" || name == "write failure" {
				assert.Zero(t, weishauptH.writes.Load())
				weishauptH.fail.Store(false)
				weishauptH.failWrites.Store(false)
			}
			if want > 0 {
				require.Eventually(t, func() bool { return weishauptH.writes.Load() >= 2 }, time.Second, time.Millisecond)
			}
			stop()

			assert.Equal(t, want, weishauptH.power.Load())
			if want == 0 {
				assert.Zero(t, weishauptH.writes.Load())
			}
		})
	}
}
