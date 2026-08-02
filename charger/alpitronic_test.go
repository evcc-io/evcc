package charger

import (
	"context"
	"net"
	"strings"
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

type hycWrite struct {
	funcCode uint8
	addr     uint16
	args     []uint16
}

// hycHandler mocks the Hypercharger's register space
type hycHandler struct {
	mbserver.RequestHandler
	input   map[uint16]uint16
	holding map[uint16]uint16
	writes  []hycWrite
}

func (h *hycHandler) HandleInputRegisters(req *mbserver.InputRegistersRequest) ([]uint16, error) {
	res := make([]uint16, 0, req.Quantity)
	for i := range req.Quantity {
		v, ok := h.input[req.Addr+i]
		if !ok {
			return nil, mbserver.ErrIllegalDataAddress
		}
		res = append(res, v)
	}
	return res, nil
}

func (h *hycHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	if req.IsWrite {
		h.writes = append(h.writes, hycWrite{req.WriteFuncCode, req.Addr, req.Args})
		for i, v := range req.Args {
			h.holding[req.Addr+uint16(i)] = v
		}
		return req.Args, nil
	}

	res := make([]uint16, 0, req.Quantity)
	for i := range req.Quantity {
		res = append(res, h.holding[req.Addr+i])
	}
	return res, nil
}

// hycReg returns the absolute address of a connector register
func hycReg(connector, reg uint16) uint16 {
	return connector*100 + reg
}

// hycRegs returns a fully populated connector input block with the given state
func hycRegs(connector, state uint16) map[uint16]uint16 {
	regs := make(map[uint16]uint16)
	for reg := range uint16(hycInputLength) {
		regs[hycReg(connector, reg)] = 0
	}
	regs[hycReg(connector, hycRegState)] = state

	return regs
}

// shared mock server: mbserver.Stop() races its accept goroutine, so the server
// is started once and never stopped; handler state is reset per test
var (
	hycOnce sync.Once
	hycURI  string
	hycSrvH = &hycHandler{RequestHandler: new(mbserver.DummyHandler)}
)

// hycTestCharger connects a charger to the shared mock Modbus server
func hycTestCharger(t *testing.T, connector uint16, regs map[uint16]uint16) (*AlpitronicHYC, *hycHandler) {
	t.Helper()

	return hycTestChargerWithLimit(t, connector, regs, nil)
}

// hycTestChargerWithLimit additionally seeds the connector's power limit
func hycTestChargerWithLimit(t *testing.T, connector uint16, regs, holding map[uint16]uint16) (*AlpitronicHYC, *hycHandler) {
	t.Helper()

	hycOnce.Do(func() {
		l, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		srv, err := mbserver.New(hycSrvH)
		require.NoError(t, err)
		require.NoError(t, srv.Start(l))

		hycURI = l.Addr().String()
	})

	if holding == nil {
		holding = make(map[uint16]uint16)
	}

	hycSrvH.input = regs
	hycSrvH.holding = holding
	hycSrvH.writes = nil

	conn, err := modbus.NewConnection(context.Background(), hycURI, "", "", 0, modbus.Tcp, 1)
	require.NoError(t, err)

	wb, err := newAlpitronicHYC(conn, connector)
	require.NoError(t, err)

	return wb, hycSrvH
}

func TestAlpitronicStatus(t *testing.T) {
	tc := []struct {
		state  uint16
		status api.ChargeStatus
		err    bool
	}{
		{hycStateAvailable, api.StatusA, false},
		{hycStatePreparingTagIdReady, api.StatusA, false}, // authorized, nothing plugged in yet
		{hycStatePreparingEVReady, api.StatusB, false},
		{hycStateCharging, api.StatusC, false},
		{hycStateSuspendedEV, api.StatusB, false},
		{hycStateSuspendedEVSE, api.StatusB, false},
		{hycStateFinishing, api.StatusB, false},
		{hycStateReserved, api.StatusA, false},
		{hycStateUnavailable, api.StatusA, false},
		{hycStateUnavailableFwUpdate, api.StatusA, false},
		{hycStateFaulted, api.StatusNone, true},
		{hycStateUnavailableConnObj, api.StatusA, false},
		{12, api.StatusNone, true},
	}

	for _, tc := range tc {
		wb, _ := hycTestCharger(t, 1, hycRegs(1, tc.state))

		status, err := wb.Status()
		if tc.err {
			assert.Error(t, err, "state %d", tc.state)
		} else {
			assert.NoError(t, err, "state %d", tc.state)
		}
		assert.Equal(t, tc.status, status, "state %d", tc.state)
	}
}

func TestAlpitronicStatusReason(t *testing.T) {
	tc := []struct {
		state  uint16
		reason api.Reason
	}{
		{hycStateAvailable, api.ReasonUnknown},
		{hycStatePreparingTagIdReady, api.ReasonUnknown},
		{hycStatePreparingEVReady, api.ReasonWaitingForAuthorization},
		{hycStateCharging, api.ReasonUnknown},
		{hycStateFinishing, api.ReasonDisconnectRequired},
	}

	for _, tc := range tc {
		wb, _ := hycTestCharger(t, 1, hycRegs(1, tc.state))

		reason, err := wb.StatusReason()
		require.NoError(t, err, "state %d", tc.state)
		assert.Equal(t, tc.reason, reason, "state %d", tc.state)
	}
}

func TestAlpitronicMeasurements(t *testing.T) {
	regs := hycRegs(1, hycStateCharging)
	regs[hycReg(1, hycRegChargingPower)+1] = 11500      // 11500 W
	regs[hycReg(1, hycRegChargeTime)] = 3600            // 1h
	regs[hycReg(1, hycRegChargedEnergy)] = 1234         // 12.34 kWh
	regs[hycReg(1, hycRegSoC)] = 6550                   // 65.5 %
	regs[hycReg(1, hycRegTotalChargedEnergy)+3] = 18705 // 18.705 kWh

	wb, _ := hycTestCharger(t, 1, regs)

	power, err := wb.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 11500.0, power)

	dur, err := wb.ChargeDuration()
	require.NoError(t, err)
	assert.Equal(t, time.Hour, dur)

	charged, err := wb.ChargedEnergy()
	require.NoError(t, err)
	assert.Equal(t, 12.34, charged)

	soc, err := wb.Soc()
	require.NoError(t, err)
	assert.Equal(t, 65.5, soc)

	total, err := wb.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 18.705, total)
}

func TestAlpitronicIdentify(t *testing.T) {
	// no vehicle id, no tag
	wb, _ := hycTestCharger(t, 1, hycRegs(1, hycStateAvailable))

	id, err := wb.Identify()
	require.NoError(t, err)
	assert.Empty(t, id)

	// vehicle id takes precedence over the tag
	regs := hycRegs(1, hycStateCharging)
	regs[hycReg(1, hycRegVID)+1] = 0xAABB
	regs[hycReg(1, hycRegVID)+2] = 0xCCDD
	regs[hycReg(1, hycRegVID)+3] = 0xEEFF
	regs[hycReg(1, hycRegIdTag)] = 0x4142

	wb, _ = hycTestCharger(t, 1, regs)

	id, err = wb.Identify()
	require.NoError(t, err)
	assert.Equal(t, "0000aabbccddeeff", id)

	// tag as fallback
	regs = hycRegs(1, hycStateCharging)
	regs[hycReg(1, hycRegIdTag)] = 0x4142

	wb, _ = hycTestCharger(t, 1, regs)

	id, err = wb.Identify()
	require.NoError(t, err)
	assert.Equal(t, "4142"+strings.Repeat("0", 36), id)
}

func TestAlpitronicEnable(t *testing.T) {
	wb, h := hycTestCharger(t, 1, hycRegs(1, hycStatePreparingEVReady))

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)

	require.NoError(t, wb.Enable(true))
	enabled, err = wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	require.NoError(t, wb.Enable(false))
	enabled, err = wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)

	// default current -> 1552W, disable -> the station must not see a zero limit
	require.Len(t, h.writes, 2)
	assert.Equal(t, hycWrite{16, hycReg(1, hycRegMaxPowerAC), []uint16{0, 1552}}, h.writes[0])
	assert.Equal(t, hycWrite{16, hycReg(1, hycRegMaxPowerAC), []uint16{0, hycMinPowerAC}}, h.writes[1])
}

func TestAlpitronicMaxCurrent(t *testing.T) {
	// current is written as power: A * 230V * 3p
	tc := []struct {
		current float64
		args    []uint16
	}{
		{hycMinCurrent, []uint16{0, 1552}}, // lowest accepted current
		{6, []uint16{0, 4140}},
		{16, []uint16{0, 11040}},
		{200, []uint16{138000 >> 16, 138000 & 0xFFFF}},
	}

	for _, tc := range tc {
		wb, h := hycTestCharger(t, 1, hycRegs(1, hycStateCharging))

		require.NoError(t, wb.MaxCurrentMillis(tc.current))
		require.Len(t, h.writes, 1, "current %v", tc.current)
		assert.Equal(t, hycWrite{16, hycReg(1, hycRegMaxPowerAC), tc.args}, h.writes[0], "current %v", tc.current)
	}
}

func TestAlpitronicSeedCurrent(t *testing.T) {
	// the default must not read back as disabled
	assert.Greater(t, hycPower(hycMinCurrent), uint32(hycMinPowerAC))

	// no usable limit set -> default
	wb, _ := hycTestCharger(t, 1, hycRegs(1, hycStateAvailable))
	assert.Equal(t, hycMinCurrent, wb.curr)

	// the disabled sentinel must not be taken over
	holding := map[uint16]uint16{hycReg(1, hycRegMaxPowerAC) + 1: hycMinPowerAC}
	wb, _ = hycTestChargerWithLimit(t, 1, hycRegs(1, hycStateAvailable), holding)
	assert.Equal(t, hycMinCurrent, wb.curr)

	// existing limit is adopted, so enabling does not fall back to the default
	holding = map[uint16]uint16{hycReg(1, hycRegMaxPowerAC) + 1: 11040}
	wb, h := hycTestChargerWithLimit(t, 1, hycRegs(1, hycStateCharging), holding)
	assert.Equal(t, 16.0, wb.curr)

	require.NoError(t, wb.Enable(true))
	require.Len(t, h.writes, 1)
	assert.Equal(t, hycWrite{16, hycReg(1, hycRegMaxPowerAC), []uint16{0, 11040}}, h.writes[0])
}

func TestAlpitronicMaxCurrentInvalid(t *testing.T) {
	wb, h := hycTestCharger(t, 1, hycRegs(1, hycStateCharging))

	// anything that does not exceed hycMinPowerAC would read back as disabled
	assert.Error(t, wb.MaxCurrentMillis(2.24))
	assert.Error(t, wb.MaxCurrentMillis(0))
	assert.Error(t, wb.MaxCurrentMillis(-1))
	assert.Empty(t, h.writes)
}

func TestAlpitronicConnector(t *testing.T) {
	// second connector reads and writes the 2xx block
	regs := hycRegs(2, hycStateCharging)
	regs[hycReg(2, hycRegChargingPower)+1] = 50000

	wb, h := hycTestCharger(t, 2, regs)

	status, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)

	power, err := wb.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 50000.0, power)

	require.NoError(t, wb.Enable(false))
	require.Len(t, h.writes, 1)
	assert.Equal(t, hycReg(2, hycRegMaxPowerAC), h.writes[0].addr)
}

func TestAlpitronicReadFailure(t *testing.T) {
	// missing register in the bulk-read block -> IllegalDataAddress propagates
	regs := hycRegs(1, hycStateCharging)
	delete(regs, hycReg(1, hycRegTotalChargedEnergy)+3)

	wb, _ := hycTestCharger(t, 1, regs)

	_, err := wb.Status()
	assert.Error(t, err)

	_, err = wb.CurrentPower()
	assert.Error(t, err)
}

func TestAlpitronicSponsorGate(t *testing.T) {
	// go-e tests set the global sponsor.Subject and never reset it
	old := sponsor.Subject
	sponsor.Subject = ""
	t.Cleanup(func() { sponsor.Subject = old })

	// tests run without sponsorship: the public constructor must refuse
	_, err := NewAlpitronicHYC(t.Context(), modbus.TcpSettings{URI: "localhost:0", ID: 1}, 1)
	assert.ErrorIs(t, err, api.ErrSponsorRequired)
}
