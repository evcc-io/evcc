package charger

import (
	"context"
	"net"
	"sync"
	"testing"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type foxWrite struct {
	funcCode uint8
	addr     uint16
	args     []uint16
}

// foxHandler mocks the charger's holding register space
type foxHandler struct {
	mbserver.RequestHandler
	mu     sync.Mutex
	regs   map[uint16]uint16
	writes []foxWrite
}

func (h *foxHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if req.IsWrite {
		h.writes = append(h.writes, foxWrite{req.WriteFuncCode, req.Addr, req.Args})
		for i, v := range req.Args {
			h.regs[req.Addr+uint16(i)] = v
		}
		return req.Args, nil
	}

	res := make([]uint16, 0, req.Quantity)
	for i := range req.Quantity {
		v, ok := h.regs[req.Addr+i]
		if !ok {
			return nil, mbserver.ErrIllegalDataAddress
		}
		res = append(res, v)
	}

	return res, nil
}

// shared mock server: mbserver.Stop() races its accept goroutine, so the server
// is started once and never stopped; handler state is reset per test
var (
	foxOnce sync.Once
	foxURI  string
	foxSrvH = &foxHandler{RequestHandler: new(mbserver.DummyHandler)}
)

// foxTestCharger returns a 22kW charger with auto phase switching connected to the mock server
func foxTestCharger(t *testing.T, regs map[uint16]uint16) (*FoxESSEVC, *foxHandler) {
	t.Helper()

	foxOnce.Do(func() {
		l, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		srv, err := mbserver.New(foxSrvH)
		require.NoError(t, err)
		require.NoError(t, srv.Start(l))

		foxURI = l.Addr().String()
	})

	foxSrvH.regs = regs
	foxSrvH.writes = nil

	conn, err := modbus.NewConnection(context.Background(), foxURI, "", "", 0, modbus.Tcp, 1)
	require.NoError(t, err)

	wb := &FoxESSEVC{
		Caps:       implement.New(),
		log:        util.NewLogger("foxess-evc"),
		conn:       conn,
		phases:     3,
		switchable: true,
		minPower:   42,
		maxPower:   220,
		minCurrent: 6,
		maxCurrent: 32,
	}

	return wb, foxSrvH
}

// fox22kW returns a 22kW charger with auto phase switching
func fox22kW(switchable bool) *FoxESSEVC {
	return &FoxESSEVC{
		phases: 3, switchable: switchable,
		minPower: 42, maxPower: 220, minCurrent: 6, maxCurrent: 32,
	}
}

// fox11kW returns an 11kW charger, rated 16A per phase
func fox11kW(switchable bool) *FoxESSEVC {
	return &FoxESSEVC{
		phases: 3, switchable: switchable,
		minPower: foxMinPower3p, maxPower: 110, minCurrent: 6, maxCurrent: 16,
	}
}

// fox7kW returns a single-phase 7.3kW charger
func fox7kW() *FoxESSEVC {
	return &FoxESSEVC{
		phases:   1,
		minPower: foxMinPower1p, maxPower: foxMaxPower1p, minCurrent: 6, maxCurrent: 32,
	}
}

func TestFoxESSEVCSetpoint(t *testing.T) {
	tc := []struct {
		name     string
		wb       *FoxESSEVC
		enabled  bool
		current  float64
		phases   int
		expected uint16
	}{
		{"disabled", fox22kW(true), false, 16, 3, 0},
		// 3 x 230V x 6A = 4.14kW rounds to 41, below the charger's 4.2kW three-phase
		// threshold - it would silently drop to single phase (§2.38)
		{"3p min", fox22kW(true), true, 6, 3, foxMinPower3p},
		{"3p below min current", fox22kW(true), true, 4, 3, foxMinPower3p},
		{"3p nominal", fox22kW(true), true, 16, 3, 110},
		{"3p max", fox22kW(true), true, 32, 3, 220},
		{"1p min", fox22kW(true), true, 6, 1, foxMinPower1p},
		{"1p at threshold", fox22kW(true), true, 18, 1, foxMinPower3p - 1},
		// without the cap the charger would deliver 7.4kW on a single phase
		{"1p max capped", fox22kW(true), true, 32, 1, foxMinPower3p - 1},
		// a charger without auto switching is bound by its device limits only
		{"fixed 3p min", fox22kW(false), true, 6, 3, foxMinPower3p},
		{"fixed 3p max", fox22kW(false), true, 32, 3, 220},
		{"1p charger min", fox7kW(), true, 6, 1, foxMinPower1p},
		{"1p charger max", fox7kW(), true, 32, 1, 73},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.expected, tc.wb.calcSetpoint(tc.enabled, tc.current, tc.phases))
		})
	}
}

func TestFoxESSEVCMinMaxCurrent(t *testing.T) {
	tc := []struct {
		name     string
		wb       *FoxESSEVC
		phases   int
		min, max float64
	}{
		{"3p switchable", fox22kW(true), 3, 6.087, 31.884},
		{"1p switchable", fox22kW(true), 1, 6.087, 17.826},
		{"3p fixed", fox22kW(false), 3, 6.087, 31.884},
		{"1p charger", fox7kW(), 1, 6.087, 31.739},
		{"11kW 3p", fox11kW(true), 3, 6.087, 15.942},
		// the 4.1kW single-phase ceiling would be 17.8A, beyond the 16A the device is rated for
		{"11kW 1p", fox11kW(true), 1, 6.087, 16},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			tc.wb.phases = tc.phases

			minCurrent, maxCurrent, err := tc.wb.GetMinMaxCurrent()
			require.NoError(t, err)
			assert.InDelta(t, tc.min, minCurrent, 0.001)
			assert.InDelta(t, tc.max, maxCurrent, 0.001)

			// the minimum current must produce a setpoint the charger accepts for that phase count
			lo, _ := tc.wb.powerLimits(tc.phases)
			assert.GreaterOrEqual(t, tc.wb.calcSetpoint(true, minCurrent, tc.phases), lo)
		})
	}
}

func TestFoxESSEVCPhases(t *testing.T) {
	wb := fox22kW(true)
	wb.phases = 1

	tc := []struct {
		setpoint uint16
		phases   int
	}{
		// below the single-phase threshold charging is paused, the tracked count applies
		{0, wb.phases},
		{foxMinPower1p - 1, wb.phases},
		{foxMinPower1p, 1},
		{foxMinPower3p - 1, 1},
		{foxMinPower3p, 3},
		{220, 3},
	}

	for _, tc := range tc {
		wb.setpoint = tc.setpoint

		phases, err := wb.getPhases()
		require.NoError(t, err)
		assert.Equal(t, tc.phases, phases, "setpoint %d", tc.setpoint)
	}
}

func TestFoxESSEVCStatus(t *testing.T) {
	tc := []struct {
		state  uint16
		status api.ChargeStatus
		err    bool
	}{
		{foxStatusIdle, api.StatusA, false},
		{foxStatusConnect, api.StatusB, false},
		{foxStatusStart, api.StatusB, false},
		{foxStatusCharging, api.StatusC, false},
		{foxStatusPause, api.StatusB, false},
		{foxStatusFinish, api.StatusB, false},
		{foxStatusFault, api.StatusNone, true},
		{7, api.StatusNone, true}, // reserved
		{foxStatusLocked, api.StatusNone, true},
		{foxStatusSwitching, api.StatusB, false},
	}

	for _, tc := range tc {
		wb, _ := foxTestCharger(t, map[uint16]uint16{
			foxRegStatus:   tc.state,
			foxRegMaxPower: 110,
		})

		status, err := wb.Status()
		if tc.err {
			assert.Error(t, err, "state %d", tc.state)
		} else {
			assert.NoError(t, err, "state %d", tc.state)
		}
		assert.Equal(t, tc.status, status, "state %d", tc.state)

		// StatusReason and GetMaxCurrent rely on the cached raw status
		assert.Equal(t, tc.state, wb.status, "state %d", tc.state)
	}
}

func TestFoxESSEVCSessionActive(t *testing.T) {
	wb := fox22kW(true)

	tc := []struct {
		state  uint16
		active bool
	}{
		{foxStatusIdle, false},
		{foxStatusConnect, false}, // not started, setpoint may still hold the restored maximum
		{foxStatusStart, true},
		{foxStatusCharging, true},
		{foxStatusPause, true}, // suspended by the car or by a zeroed setpoint
		{foxStatusFinish, false},
		{foxStatusFault, false},
		{foxStatusLocked, false},
		{foxStatusSwitching, true},
	}

	for _, tc := range tc {
		assert.Equal(t, tc.active, wb.sessionActive(tc.state), "state %d", tc.state)
	}
}

func TestFoxESSEVCStatusReason(t *testing.T) {
	wb := fox22kW(true)

	wb.status = foxStatusConnect
	reason, err := wb.StatusReason()
	require.NoError(t, err)
	assert.Equal(t, api.ReasonWaitingForAuthorization, reason)

	wb.status = foxStatusFinish
	reason, err = wb.StatusReason()
	require.NoError(t, err)
	assert.Equal(t, api.ReasonDisconnectRequired, reason)

	wb.status = foxStatusCharging
	reason, err = wb.StatusReason()
	require.NoError(t, err)
	assert.Equal(t, api.ReasonUnknown, reason)
}

func TestFoxESSEVCEnable(t *testing.T) {
	wb, h := foxTestCharger(t, map[uint16]uint16{
		foxRegStatus:   foxStatusCharging,
		foxRegMaxPower: 0,
	})
	wb.current = 16

	require.NoError(t, wb.Enable(true))
	require.Len(t, h.writes, 1)
	assert.Equal(t, uint16(foxRegMaxPower), h.writes[0].addr)
	assert.Equal(t, []uint16{110}, h.writes[0].args)

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	// an unchanged limit is rewritten as well, it keeps the validity window alive (§2.34)
	require.NoError(t, wb.MaxCurrentMillis(16))
	require.Len(t, h.writes, 2)
	assert.Equal(t, []uint16{110}, h.writes[1].args)

	require.NoError(t, wb.Enable(false))
	require.Len(t, h.writes, 3)
	assert.Equal(t, []uint16{0}, h.writes[2].args)

	enabled, err = wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestFoxESSEVCPhaseSwitch(t *testing.T) {
	wb, h := foxTestCharger(t, map[uint16]uint16{
		foxRegStatus:   foxStatusCharging,
		foxRegMaxPower: 110,
	})
	wb.current = 16
	wb.enabled = true

	// switching to single phase must rewrite the setpoint right away- 230V x 16A stays below
	// the three-phase threshold, so the charger settles on single phase
	require.NoError(t, wb.phases1p3p(1))
	require.Len(t, h.writes, 1)
	assert.Equal(t, []uint16{37}, h.writes[0].args)
	assert.Equal(t, 1, wb.phases)

	phases, err := wb.getPhases()
	require.NoError(t, err)
	assert.Equal(t, 1, phases)
}

// TestFoxESSEVCConcurrent exercises the tracked state from several goroutines, as the device
// status API does while the loadpoint and the heartbeat are running. Meaningful under -race.
func TestFoxESSEVCConcurrent(t *testing.T) {
	wb, _ := foxTestCharger(t, map[uint16]uint16{
		foxRegStatus:   foxStatusCharging,
		foxRegMaxPower: 110,
	})
	wb.current = 16
	wb.enabled = true

	var wg sync.WaitGroup
	for range 8 {
		wg.Go(func() {
			for range 20 {
				_, _ = wb.Status()
				_, _ = wb.StatusReason()
				_, _ = wb.Enabled()
				_, _ = wb.GetMaxCurrent()
				_, _, _ = wb.GetMinMaxCurrent()
				_, _ = wb.getPhases()
				_ = wb.MaxCurrentMillis(16)
				_ = wb.phases1p3p(1)
				_ = wb.Enable(true)

				// heartbeat
				wb.mu.Lock()
				_ = wb.writeReg(foxRegMaxPower, wb.setpoint)
				wb.mu.Unlock()
			}
		})
	}

	wg.Wait()
}

func TestFoxESSEVCGetMaxCurrent(t *testing.T) {
	wb, _ := foxTestCharger(t, map[uint16]uint16{
		foxRegMaxPower: 110,
	})

	// outside an active session the charger reports its restored max supported power (§2.31)
	wb.status = foxStatusFinish
	_, err := wb.GetMaxCurrent()
	assert.ErrorIs(t, err, api.ErrNotAvailable)

	wb.status = foxStatusCharging
	current, err := wb.GetMaxCurrent()
	require.NoError(t, err)
	assert.InDelta(t, 15.942, current, 0.001)

	// the setpoint encodes the phase count, so a three-phase setpoint read back while the tracked
	// count is 1p resolves to 3p rather than the impossible 11kW/230V = 47.8A
	wb.phases = 1
	current, err = wb.GetMaxCurrent()
	require.NoError(t, err)
	assert.InDelta(t, 15.942, current, 0.001)
}
