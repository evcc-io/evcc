package charger

import (
	"context"
	"encoding/binary"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hagerHandler mocks the witty plus register space
type hagerHandler struct {
	mbserver.RequestHandler
	regs   map[uint16]uint16
	writes map[uint16]uint16
}

func (h *hagerHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	if req.IsWrite {
		for i, v := range req.Args {
			h.writes[req.Addr+uint16(i)] = v
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

// shared mock server: mbserver.Stop() races its accept goroutine, so the server is
// started once and never stopped; handler state is reset per test
var (
	hagerOnce sync.Once
	hagerURI  string
	hagerSrvH = &hagerHandler{RequestHandler: new(mbserver.DummyHandler)}
)

// hagerRegs returns a minimal register set for a charger in the given CP state
func hagerRegs(cpState uint16) map[uint16]uint16 {
	return map[uint16]uint16{
		hagerRegCpState:      cpState,
		hagerRegMaxCurrent:   0,
		hagerRegAvailability: hagerAvailable,
		hagerRegSwitch3to1:   0,
		hagerRegHwMaxCurrent: 32,
	}
}

// hagerTestCharger connects a charger to the shared mock Modbus server
func hagerTestCharger(t *testing.T, regs map[uint16]uint16) (*HagerWitty, *hagerHandler) {
	t.Helper()

	hagerOnce.Do(func() {
		l, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		srv, err := mbserver.New(hagerSrvH)
		require.NoError(t, err)
		require.NoError(t, srv.Start(l))

		hagerURI = l.Addr().String()
	})

	hagerSrvH.regs = regs
	hagerSrvH.writes = make(map[uint16]uint16)

	conn, err := modbus.NewConnection(context.Background(), hagerURI, "", "", 0, modbus.Tcp, 1)
	require.NoError(t, err)

	wb := newHagerWitty(conn, util.NewLogger("hager"))
	require.NoError(t, wb.initialize())

	return wb, hagerSrvH
}

// hagerAscii encodes a two-character CP state as a single register
func hagerAscii(s string) uint16 {
	return binary.BigEndian.Uint16([]byte(s))
}

func TestHagerWittyStatusAscii(t *testing.T) {
	tc := []struct {
		cp     string
		status api.ChargeStatus
		err    bool
	}{
		{"A1", api.StatusA, false},
		{"B1", api.StatusB, false},
		{"B2", api.StatusB, false},
		{"C1", api.StatusB, false},
		{"C2", api.StatusC, false},
		{"E ", api.ChargeStatus("E"), true},
		{"F ", api.ChargeStatus("F"), true},
	}

	for _, tc := range tc {
		wb, _ := hagerTestCharger(t, hagerRegs(hagerAscii(tc.cp)))
		require.True(t, wb.cpAscii, tc.cp)

		status, err := wb.Status()
		if tc.err {
			assert.Error(t, err, tc.cp)
		} else {
			assert.NoError(t, err, tc.cp)
		}
		assert.Equal(t, tc.status, status, tc.cp)
	}
}

func TestHagerWittyStatusNumeric(t *testing.T) {
	tc := []struct {
		state  uint32
		status api.ChargeStatus
		err    bool
	}{
		{hagerStateIdle, api.StatusA, false},
		{hagerStateWait, api.StatusB, false},
		{hagerStateWaitEnergy, api.StatusB, false},
		{hagerStateCharging, api.StatusC, false},
		{hagerStateFinished, api.StatusB, false},
		{hagerStateReserved, api.StatusB, false},
		{hagerStateError, api.StatusNone, true},
	}

	for _, tc := range tc {
		regs := hagerRegs(1) // numeric CP state
		raw := tc.state << 8
		regs[hagerRegStatus] = uint16(raw >> 16)
		regs[hagerRegStatus+1] = uint16(raw)

		wb, _ := hagerTestCharger(t, regs)
		require.False(t, wb.cpAscii, tc.state)

		status, err := wb.Status()
		if tc.err {
			assert.Error(t, err, tc.state)
		} else {
			assert.NoError(t, err, tc.state)
		}
		assert.Equal(t, tc.status, status, tc.state)
	}

	// waiting for authorization is reported as status reason
	regs := hagerRegs(1)
	regs[hagerRegStatus] = 0
	regs[hagerRegStatus+1] = uint16(hagerStateWaitEnergy << 8)

	wb, _ := hagerTestCharger(t, regs)
	_, err := wb.Status()
	require.NoError(t, err)

	reason, err := wb.StatusReason()
	require.NoError(t, err)
	assert.Equal(t, api.ReasonWaitingForAuthorization, reason)
}

func TestHagerWittyEnable(t *testing.T) {
	wb, h := hagerTestCharger(t, hagerRegs(hagerAscii("A1")))

	// availability is asserted on startup
	assert.Equal(t, uint16(hagerAvailable), h.writes[hagerRegAvailability])

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)

	// current is remembered while disabled
	require.NoError(t, wb.MaxCurrent(16))
	assert.NotContains(t, h.writes, hagerRegMaxCurrent)

	require.NoError(t, wb.Enable(true))
	assert.Equal(t, uint16(160), h.writes[hagerRegMaxCurrent])

	enabled, err = wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	current, err := wb.GetMaxCurrent()
	require.NoError(t, err)
	assert.Equal(t, 16.0, current)

	require.NoError(t, wb.MaxCurrentMillis(6.5))
	assert.Equal(t, uint16(65), h.writes[hagerRegMaxCurrent])

	// the setpoint is clamped to the device maximum
	require.NoError(t, wb.MaxCurrent(40))
	assert.Equal(t, uint16(hagerMaxCurrent), h.writes[hagerRegMaxCurrent])

	require.Error(t, wb.MaxCurrent(5))

	require.NoError(t, wb.Enable(false))
	assert.Equal(t, uint16(0), h.writes[hagerRegMaxCurrent])

	enabled, err = wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestHagerWittyEnableDefaultsToMinCurrent(t *testing.T) {
	wb, h := hagerTestCharger(t, hagerRegs(hagerAscii("B2")))

	// enabling without a preceding current command must not command the device maximum
	require.NoError(t, wb.Enable(true))
	assert.Equal(t, uint16(hagerMinCurrent), h.writes[hagerRegMaxCurrent])
}

func TestHagerWittyEnabledOnStartup(t *testing.T) {
	regs := hagerRegs(hagerAscii("C2"))
	regs[hagerRegMaxCurrent] = 100

	wb, _ := hagerTestCharger(t, regs)

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	// disabling and re-enabling restores the running setpoint
	require.NoError(t, wb.Enable(false))
	require.NoError(t, wb.Enable(true))

	current, err := wb.GetMaxCurrent()
	require.NoError(t, err)
	assert.Equal(t, 10.0, current)
}

func TestHagerWittyPhases(t *testing.T) {
	wb, h := hagerTestCharger(t, hagerRegs(hagerAscii("B2")))

	ps, ok := api.Cap[api.PhaseSwitcher](wb)
	require.True(t, ok)
	pg, ok := api.Cap[api.PhaseGetter](wb)
	require.True(t, ok)

	require.NoError(t, ps.Phases1p3p(1))
	assert.Equal(t, uint16(1), h.writes[hagerRegSwitch3to1])

	phases, err := pg.GetPhases()
	require.NoError(t, err)
	assert.Equal(t, 1, phases)

	require.NoError(t, ps.Phases1p3p(3))
	assert.Equal(t, uint16(0), h.writes[hagerRegSwitch3to1])

	phases, err = pg.GetPhases()
	require.NoError(t, err)
	assert.Equal(t, 3, phases)
}

func TestHagerWittyMeasurements(t *testing.T) {
	regs := hagerRegs(hagerAscii("C2"))
	regs[hagerRegCurrents] = 100  // 10.0A
	regs[hagerRegCurrents+1] = 99 // 9.9A
	regs[hagerRegCurrents+2] = 0
	regs[hagerRegVoltages] = 2300 // 230.0V
	regs[hagerRegVoltages+1] = 2301
	regs[hagerRegVoltages+2] = 2302
	regs[hagerRegPowers] = 2300
	regs[hagerRegPowers+1] = 2277
	regs[hagerRegPowers+2] = 0
	regs[hagerRegPower] = 0
	regs[hagerRegPower+1] = 4577
	regs[hagerRegSessionEnergy] = 0
	regs[hagerRegSessionEnergy+1] = 1500 // 1.5kWh
	regs[hagerRegSessionDuration] = 0
	regs[hagerRegSessionDuration+1] = 1800
	regs[hagerRegRfidSize] = 4
	regs[hagerRegRfidUid] = 0x1234
	regs[hagerRegRfidUid+1] = 0xABCD
	for i := uint16(2); i < 5; i++ {
		regs[hagerRegRfidUid+i] = 0
	}

	wb, _ := hagerTestCharger(t, regs)

	power, err := wb.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 4577.0, power)

	i1, i2, i3, err := wb.Currents()
	require.NoError(t, err)
	assert.Equal(t, []float64{10, 9.9, 0}, []float64{i1, i2, i3})

	u1, u2, u3, err := wb.Voltages()
	require.NoError(t, err)
	assert.Equal(t, []float64{230, 230.1, 230.2}, []float64{u1, u2, u3})

	p1, p2, p3, err := wb.Powers()
	require.NoError(t, err)
	assert.Equal(t, []float64{2300, 2277, 0}, []float64{p1, p2, p3})

	energy, err := wb.ChargedEnergy()
	require.NoError(t, err)
	assert.Equal(t, 1.5, energy)

	dur, err := wb.ChargeDuration()
	require.NoError(t, err)
	assert.Equal(t, 30*time.Minute, dur)

	id, err := wb.Identify()
	require.NoError(t, err)
	assert.Equal(t, []string{"1234ABCD"}, id)

	cl, ok := api.Cap[api.CurrentLimiter](wb)
	require.True(t, ok)

	minCurrent, maxCurrent, err := cl.GetMinMaxCurrent()
	require.NoError(t, err)
	assert.Equal(t, []float64{6, 32}, []float64{minCurrent, maxCurrent})
}
