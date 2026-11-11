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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type polWrite struct {
	addr, val uint16
}

// polHandler mocks the Policharger holding register space
type polHandler struct {
	mbserver.RequestHandler
	mu     sync.Mutex
	regs   map[uint16]uint16
	writes []polWrite
}

func (h *polHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if req.IsWrite {
		for i, v := range req.Args {
			addr := req.Addr + uint16(i)
			// the charger rejects 0xFF00 for booleans
			if v == 0xFF00 && (addr == polRegLock || addr == polRegLockType) {
				return nil, mbserver.ErrIllegalDataValue
			}
			h.regs[addr] = v
			h.writes = append(h.writes, polWrite{addr, v})
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

// polRegsIdle returns a locked, armed charger with a car connected
func polRegsIdle() map[uint16]uint16 {
	regs := make(map[uint16]uint16)
	for reg := uint16(1); reg <= 95; reg++ {
		regs[reg] = 0
	}
	regs[polRegModel] = 4
	regs[polRegConnector] = 2
	regs[polRegSwVersion] = 9
	regs[polRegSwVersion+1] = 30
	regs[polRegLockType] = 1
	regs[polRegManualLimit] = 6
	regs[polRegPvGridLimit] = 16
	regs[polRegSetpoint] = polArmCurrent
	regs[polRegPvProgram] = 1
	regs[polRegPinConnected] = 1
	regs[polRegLock] = 1
	return regs
}

var (
	polOnce sync.Once
	polURI  string
	polSrvH = &polHandler{RequestHandler: new(mbserver.DummyHandler)}
)

// polTestCharger connects a charger to the shared mock Modbus server
func polTestCharger(t *testing.T, regs map[uint16]uint16) (*Policharger, *polHandler) {
	t.Helper()

	polOnce.Do(func() {
		l, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		srv, err := mbserver.New(polSrvH)
		require.NoError(t, err)
		require.NoError(t, srv.Start(l))

		polURI = l.Addr().String()
	})

	polSrvH.mu.Lock()
	polSrvH.regs = regs
	polSrvH.writes = nil
	polSrvH.mu.Unlock()

	conn, err := modbus.NewConnection(context.Background(), polURI, "", "", 0, modbus.Tcp, 1)
	require.NoError(t, err)

	// no read cache: every call sees the current mock registers
	return newPolicharger(conn, util.NewLogger("test"), 0), polSrvH
}

func (h *polHandler) written() []polWrite {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]polWrite(nil), h.writes...)
}

func (h *polHandler) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.writes = nil
}

func TestPolichargerStatus(t *testing.T) {
	tc := []struct {
		name   string
		mod    func(map[uint16]uint16)
		status api.ChargeStatus
	}{
		{"locked, connected", func(r map[uint16]uint16) {}, api.StatusB},
		{"locked, disconnected", func(r map[uint16]uint16) { r[polRegPinConnected] = 0 }, api.StatusA},
		{"connected via reg 39 only", func(r map[uint16]uint16) {
			r[polRegPinConnected] = 0
			r[polRegConnected] = 1
		}, api.StatusB},
		{"charging", func(r map[uint16]uint16) {
			r[polRegLock] = 0
			r[polRegCharging] = 1
			r[polRegCurrent] = 12
		}, api.StatusC},
	}

	for _, tc := range tc {
		regs := polRegsIdle()
		tc.mod(regs)
		wb, _ := polTestCharger(t, regs)

		status, err := wb.Status()
		require.NoError(t, err, tc.name)
		assert.Equal(t, tc.status, status, tc.name)
	}
}

func TestPolichargerStatusArmsWhenIdle(t *testing.T) {
	regs := polRegsIdle()
	regs[polRegSetpoint] = 9 // stopped after charging below 16 A
	regs[polRegManualLimit] = 16
	wb, h := polTestCharger(t, regs)
	wb.current = 9

	status, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusB, status)

	// cap first, then arm
	assert.Equal(t, []polWrite{{polRegManualLimit, 9}, {polRegSetpoint, polArmCurrent}}, h.written())
	assert.True(t, wb.armed)

	// nothing left to do on the next cycle
	h.reset()
	_, err = wb.Status()
	require.NoError(t, err)
	assert.Empty(t, h.written())
}

func TestPolichargerNeverArmsInNormalMode(t *testing.T) {
	regs := polRegsIdle()
	regs[polRegLockType] = 0 // e.g. after a reset of the charger
	regs[polRegLock] = 0
	regs[polRegSetpoint] = 0
	wb, h := polTestCharger(t, regs)

	_, err := wb.Status()
	require.NoError(t, err)

	// locked and back in normally locked mode before reg 80 = 16
	assert.Equal(t, []polWrite{
		{polRegLock, 1},
		{polRegLockType, 1},
		{polRegSetpoint, polArmCurrent},
	}, h.written())
}

func TestPolichargerArmChecksLockMode(t *testing.T) {
	regs := polRegsIdle()
	regs[polRegLockType] = 0
	regs[polRegLock] = 0
	regs[polRegSetpoint] = 0
	wb, h := polTestCharger(t, regs)

	// periodic check not due yet: the check before arming must catch it
	wb.cfgOK = time.Now().Add(-2 * time.Minute)

	_, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, []polWrite{
		{polRegLock, 1},
		{polRegLockType, 1},
		{polRegSetpoint, polArmCurrent},
	}, h.written())
}

func TestPolichargerEnable(t *testing.T) {
	wb, h := polTestCharger(t, polRegsIdle())
	wb.current = 10

	require.NoError(t, wb.Enable(true))
	assert.Equal(t, []polWrite{{polRegManualLimit, 10}, {polRegLock, 0}}, h.written())

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)
}

func TestPolichargerDisable(t *testing.T) {
	regs := polRegsIdle()
	regs[polRegLock] = 0
	regs[polRegCharging] = 1
	regs[polRegSetpoint] = 12
	regs[polRegManualLimit] = 12
	wb, h := polTestCharger(t, regs)
	wb.current = 12

	// the mock does not stop charging by itself
	require.NoError(t, wb.Enable(false))
	assert.Equal(t, polWrite{polRegLock, 1}, h.written()[0], "lock must be the first write")

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestPolichargerDisableRearms(t *testing.T) {
	regs := polRegsIdle()
	regs[polRegLock] = 0
	regs[polRegSetpoint] = 12
	regs[polRegManualLimit] = 12
	wb, h := polTestCharger(t, regs)
	wb.current = 12

	require.NoError(t, wb.Enable(false))
	assert.Equal(t, []polWrite{{polRegLock, 1}, {polRegSetpoint, polArmCurrent}}, h.written())
	assert.True(t, wb.armed)
}

func TestPolichargerMaxCurrentCharging(t *testing.T) {
	regs := polRegsIdle()
	regs[polRegLock] = 0
	regs[polRegCharging] = 1
	regs[polRegSetpoint] = 12
	regs[polRegManualLimit] = 12
	wb, h := polTestCharger(t, regs)

	// lower: setpoint only
	require.NoError(t, wb.MaxCurrent(9))
	assert.Equal(t, []polWrite{{polRegSetpoint, 9}}, h.written())

	// above the cap: setpoint first, then lift the cap
	h.reset()
	require.NoError(t, wb.MaxCurrent(14))
	assert.Equal(t, []polWrite{{polRegSetpoint, 14}, {polRegManualLimit, 14}}, h.written())
	assert.False(t, wb.armed)
}

func TestPolichargerMaxCurrentIdle(t *testing.T) {
	wb, h := polTestCharger(t, polRegsIdle())

	// idle: only the cap changes, reg 80 stays armed
	require.NoError(t, wb.MaxCurrent(11))
	assert.Equal(t, []polWrite{{polRegManualLimit, 11}}, h.written())

	assert.Error(t, wb.MaxCurrent(5))
	assert.Error(t, wb.MaxCurrent(33))
}

func TestPolichargerPvGridLimit(t *testing.T) {
	regs := polRegsIdle()
	regs[polRegPvGridLimit] = 0 // blocks grid charging with the PV licence
	wb, h := polTestCharger(t, regs)

	_, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, []polWrite{{polRegPvGridLimit, polArmCurrent}}, h.written())
}

func TestPolichargerMeasurements(t *testing.T) {
	regs := polRegsIdle()
	regs[polRegLock] = 0
	regs[polRegCharging] = 1
	regs[polRegCurrent] = 13
	wb, _ := polTestCharger(t, regs)

	power, err := wb.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 2990.0, power)

	l1, l2, l3, err := wb.Currents()
	require.NoError(t, err)
	assert.Equal(t, []float64{13, 0, 0}, []float64{l1, l2, l3})

	// not charging: no power even if the sensor reads a residual value
	regs = polRegsIdle()
	regs[polRegCurrent] = 1
	wb, _ = polTestCharger(t, regs)

	power, err = wb.CurrentPower()
	require.NoError(t, err)
	assert.Zero(t, power)
}
