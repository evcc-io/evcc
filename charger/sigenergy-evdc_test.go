package charger

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type evdcWrite struct {
	funcCode uint8
	addr     uint16
	args     []uint16
}

// evdcHandler mocks the hybrid inverter's EVDC register space
type evdcHandler struct {
	mbserver.RequestHandler
	regs       map[uint16]uint16
	writes     []evdcWrite
	failWrites bool
}

func (h *evdcHandler) HandleInputRegisters(req *mbserver.InputRegistersRequest) ([]uint16, error) {
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

func (h *evdcHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	if !req.IsWrite {
		return nil, mbserver.ErrIllegalDataAddress
	}
	if h.failWrites {
		return nil, mbserver.ErrIllegalDataValue
	}
	h.writes = append(h.writes, evdcWrite{req.WriteFuncCode, req.Addr, req.Args})
	return req.Args, nil
}

// evdcRegs returns a fully populated 31500-31520 block with the given running state
func evdcRegs(state uint16) map[uint16]uint16 {
	regs := make(map[uint16]uint16)
	for reg := uint16(evdcBase); reg < evdcBase+evdcInputLen; reg++ {
		regs[reg] = 0
	}
	regs[evdcRegRunningState] = state
	return regs
}

// shared mock server: mbserver.Stop() races its accept goroutine (unlocked
// tcpListener read vs. nil assignment), so the server is started once and
// never stopped; handler state is reset per test
var (
	evdcOnce sync.Once
	evdcURI  string
	evdcSrvH = &evdcHandler{RequestHandler: new(mbserver.DummyHandler)}
)

// evdcTestCharger connects a charger to the shared mock Modbus server
func evdcTestCharger(t *testing.T, regs map[uint16]uint16) (*SigenergyEVDC, *evdcHandler) {
	t.Helper()

	evdcOnce.Do(func() {
		l, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		srv, err := mbserver.New(evdcSrvH)
		require.NoError(t, err)
		require.NoError(t, srv.Start(l))

		evdcURI = l.Addr().String()
	})

	evdcSrvH.regs = regs
	evdcSrvH.writes = nil
	evdcSrvH.failWrites = false

	conn, err := modbus.NewConnection(context.Background(), evdcURI, "", "", 0, modbus.Tcp, 1)
	require.NoError(t, err)

	wb := newSigenergyEVDC(conn)
	wb.ratedPower = 25000

	return wb, evdcSrvH
}

func TestSigenergyEVDCStatus(t *testing.T) {
	tc := []struct {
		state  uint16
		status api.ChargeStatus
		err    bool
	}{
		{evdcStateIdle, api.StatusA, false},
		{evdcStateOccupied, api.StatusB, false},
		{evdcStatePreparing, api.StatusB, false},
		{evdcStateCharging, api.StatusC, false},
		{evdcStateScheduled, api.StatusB, false},
		{evdcStateEnded, api.StatusB, false},
		{evdcStateInsulation, api.StatusB, false},
		{evdcStateDischarging, api.StatusC, false},
		{evdcStateFault, api.StatusNone, true},
		{evdcStateUnavailable, api.StatusNone, true},
		{evdcStateAlarm, api.StatusNone, true},
		{0x0B, api.StatusNone, true},
	}

	for _, tc := range tc {
		wb, _ := evdcTestCharger(t, evdcRegs(tc.state))

		status, err := wb.Status()
		if tc.err {
			assert.Error(t, err, "state %d", tc.state)
		} else {
			assert.NoError(t, err, "state %d", tc.state)
		}
		assert.Equal(t, tc.status, status, "state %d", tc.state)
	}
}

func TestSigenergyEVDCMeasurements(t *testing.T) {
	regs := evdcRegs(evdcStateCharging)
	regs[evdcRegOutputPower] = 0
	regs[evdcRegOutputPower+1] = 11500 // 11500 W
	regs[evdcRegVehicleSoc] = 655      // 65.5 %
	regs[evdcRegSessionEnergy] = 0
	regs[evdcRegSessionEnergy+1] = 1234 // 12.34 kWh
	regs[evdcRegSessionDuration] = 0
	regs[evdcRegSessionDuration+1] = 3600 // 1h
	regs[evdcRegTotalEnergy] = 1
	regs[evdcRegTotalEnergy+1] = 34464 // 65536+34464 = 100000 -> 1000.00 kWh
	regs[evdcRegTotalDischargeEnergy] = 0
	regs[evdcRegTotalDischargeEnergy+1] = 500 // 5.00 kWh

	wb, _ := evdcTestCharger(t, regs)

	power, err := wb.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 11500.0, power)

	soc, err := wb.Soc()
	require.NoError(t, err)
	assert.Equal(t, 65.5, soc)

	charged, err := wb.ChargedEnergy()
	require.NoError(t, err)
	assert.Equal(t, 12.34, charged)

	dur, err := wb.ChargeDuration()
	require.NoError(t, err)
	assert.Equal(t, time.Hour, dur)

	total, err := wb.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 1000.0, total)

	returned, err := wb.ReturnEnergy()
	require.NoError(t, err)
	assert.Equal(t, 5.0, returned)
}

func TestSigenergyEVDCNegativePower(t *testing.T) {
	regs := evdcRegs(evdcStateDischarging)
	regs[evdcRegOutputPower] = 0xFFFF
	regs[evdcRegOutputPower+1] = 0xEC78 // -5000 W as S32

	wb, _ := evdcTestCharger(t, regs)

	power, err := wb.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, -5000.0, power)
}

func TestSigenergyEVDCSocNotAvailable(t *testing.T) {
	wb, _ := evdcTestCharger(t, evdcRegs(evdcStateIdle)) // all-zero block

	_, err := wb.Soc()
	assert.ErrorIs(t, err, api.ErrNotAvailable)
}

func TestSigenergyEVDCEnable(t *testing.T) {
	wb, h := evdcTestCharger(t, evdcRegs(evdcStateOccupied))

	require.NoError(t, wb.Enable(true))
	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	require.NoError(t, wb.Enable(false))
	enabled, err = wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)

	// start/stop is a single-register FC06 write to 41000: 0=start, 1=stop
	require.Len(t, h.writes, 2)
	assert.Equal(t, evdcWrite{6, evdcRegStartStop, []uint16{0}}, h.writes[0])
	assert.Equal(t, evdcWrite{6, evdcRegStartStop, []uint16{1}}, h.writes[1])
}

func TestSigenergyEVDCEnableWriteFailure(t *testing.T) {
	wb, h := evdcTestCharger(t, evdcRegs(evdcStateOccupied))
	h.failWrites = true

	assert.Error(t, wb.Enable(true))

	// failed write must not flip the cached state
	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestSigenergyEVDCEnabledSync(t *testing.T) {
	// charging syncs the cache to enabled
	wb, _ := evdcTestCharger(t, evdcRegs(evdcStateCharging))
	_, err := wb.Status()
	require.NoError(t, err)
	enabled, _ := wb.Enabled()
	assert.True(t, enabled)

	// ambiguous lifecycle state leaves the cache untouched
	wb, _ = evdcTestCharger(t, evdcRegs(evdcStateEnded))
	wb.enabled = true
	_, err = wb.Status()
	require.NoError(t, err)
	enabled, _ = wb.Enabled()
	assert.True(t, enabled)

	// discharging must NOT sync (vendor-initiated V2X, evcc-disabled loadpoint stays consistent)
	wb, _ = evdcTestCharger(t, evdcRegs(evdcStateDischarging))
	_, err = wb.Status()
	require.NoError(t, err)
	enabled, _ = wb.Enabled()
	assert.False(t, enabled)
}

func TestSigenergyEVDCMaxCurrent(t *testing.T) {
	tc := []struct {
		current float64
		power   uint16 // expected low word; high word is 0 in all cases
	}{
		{6, 4140},
		{16, 11040},
		{1.45, 1000}, // 1000.5 W truncated to 1000
		{1.0, 690},   // minimum current -> 690 W
		{40, 25000},  // 27600 W clamped down to rated power
	}

	for _, tc := range tc {
		wb, h := evdcTestCharger(t, evdcRegs(evdcStateCharging))

		require.NoError(t, wb.MaxCurrentMillis(tc.current))
		require.Len(t, h.writes, 1, "current %v", tc.current)

		w := h.writes[0]
		assert.Equal(t, uint8(16), w.funcCode, "current %v", tc.current)
		assert.Equal(t, uint16(evdcRegPowerLimit), w.addr, "current %v", tc.current)
		assert.Equal(t, []uint16{0, tc.power}, w.args, "current %v", tc.current)
	}
}

func TestSigenergyEVDCMaxCurrentInvalid(t *testing.T) {
	wb, h := evdcTestCharger(t, evdcRegs(evdcStateCharging))

	assert.Error(t, wb.MaxCurrentMillis(0.9))
	assert.Error(t, wb.MaxCurrentMillis(0))
	assert.Error(t, wb.MaxCurrentMillis(-1))
	assert.Empty(t, h.writes)
}

func TestSigenergyEVDCMinMaxCurrent(t *testing.T) {
	wb, _ := evdcTestCharger(t, evdcRegs(evdcStateIdle))

	minA, maxA, err := wb.GetMinMaxCurrent()
	require.NoError(t, err)
	assert.InDelta(t, 1.0, minA, 0.001)  // evdcMinCurrent
	assert.InDelta(t, 36.23, maxA, 0.01) // 25000 W / 690
}

func TestSigenergyEVDCSponsorGate(t *testing.T) {
	// go-e tests set the global sponsor.Subject and never reset it
	old := sponsor.Subject
	sponsor.Subject = ""
	t.Cleanup(func() { sponsor.Subject = old })

	// tests run without sponsorship: the public constructor must refuse
	_, err := NewSigenergyEVDC(t.Context(), "localhost:0", 1)
	assert.ErrorIs(t, err, api.ErrSponsorRequired)
}

func TestSigenergyEVDCReadFailure(t *testing.T) {
	// missing register in the bulk-read block -> IllegalDataAddress propagates
	regs := evdcRegs(evdcStateCharging)
	delete(regs, uint16(evdcRegRunningState))

	wb, _ := evdcTestCharger(t, regs)

	_, err := wb.Status()
	assert.Error(t, err)

	_, err = wb.CurrentPower()
	assert.Error(t, err)
}

func TestSigenergyEVDCWakeUp(t *testing.T) {
	wb, h := evdcTestCharger(t, evdcRegs(evdcStateOccupied))

	// realistic scenario: cache latched true from a prior session, external stop happened
	wb.enabled = true

	require.NoError(t, wb.WakeUp())

	// single FC06 start write, must not touch the enabled cache
	require.Len(t, h.writes, 1)
	assert.Equal(t, evdcWrite{6, evdcRegStartStop, []uint16{0}}, h.writes[0])

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)
}
