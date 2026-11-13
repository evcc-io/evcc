package charger

import (
	"net"
	"sync"
	"testing"

	"github.com/andig/mbserver"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type polWrite struct{ addr, val uint16 }

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
			h.regs[req.Addr+uint16(i)] = v
			h.writes = append(h.writes, polWrite{req.Addr + uint16(i), v})
		}
		return req.Args, nil
	}

	res := make([]uint16, 0, req.Quantity)
	for i := range req.Quantity {
		res = append(res, h.regs[req.Addr+i])
	}
	return res, nil
}

func (h *polHandler) written() []polWrite {
	h.mu.Lock()
	defer h.mu.Unlock()
	w := h.writes
	h.writes = nil
	return w
}

// polMux serves one mock charger per slave id from a single server, since
// mbserver.Stop races its accept loop
type polMux struct {
	mbserver.RequestHandler
	mu       sync.Mutex
	handlers map[uint8]*polHandler
}

func (m *polMux) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	m.mu.Lock()
	h := m.handlers[req.UnitId]
	m.mu.Unlock()
	return h.HandleHoldingRegisters(req)
}

var (
	polOnce sync.Once
	polURI  string
	polSrv  = &polMux{RequestHandler: new(mbserver.DummyHandler), handlers: make(map[uint8]*polHandler)}
)

// polCharger starts a mock charger, locked and armed with a car plugged in, applies mod and connects to it
func polCharger(t *testing.T, mod func(map[uint16]uint16)) (*Policharger, *polHandler, error) {
	t.Helper()

	polOnce.Do(func() {
		l, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)
		srv, err := mbserver.New(polSrv)
		require.NoError(t, err)
		require.NoError(t, srv.Start(l))
		polURI = l.Addr().String()
	})

	regs := map[uint16]uint16{
		polRegLockType:   1,
		polRegLimit:      6,
		polRegSetpoint:   polArmCurrent,
		polRegPinPlugged: 1,
		polRegLock:       1,
	}
	if mod != nil {
		mod(regs)
	}

	h := &polHandler{RequestHandler: new(mbserver.DummyHandler), regs: regs}

	polSrv.mu.Lock()
	id := uint8(len(polSrv.handlers) + 1)
	polSrv.handlers[id] = h
	polSrv.mu.Unlock()

	c, err := NewPolicharger(t.Context(), modbus.Settings{URI: polURI, ID: id})
	if err != nil {
		return nil, h, err
	}

	return c.(*Policharger), h, nil
}

func TestPolichargerNew(t *testing.T) {
	t.Run("normal mode", func(t *testing.T) {
		_, h, err := polCharger(t, func(r map[uint16]uint16) { r[polRegLockType] = 0 })
		assert.ErrorContains(t, err, "lock type")
		assert.Empty(t, h.written())
	})

	t.Run("idle and unlocked", func(t *testing.T) {
		_, h, err := polCharger(t, func(r map[uint16]uint16) { r[polRegLock] = 0 })
		require.NoError(t, err)
		assert.Equal(t, []polWrite{{polRegLock, 1}}, h.written())
	})

	t.Run("charging", func(t *testing.T) {
		_, h, err := polCharger(t, func(r map[uint16]uint16) {
			r[polRegLock] = 0
			r[polRegCharging] = 1
		})
		require.NoError(t, err)
		assert.Empty(t, h.written())
	})

	t.Run("three-phase", func(t *testing.T) {
		wb, _, err := polCharger(t, func(r map[uint16]uint16) { r[polRegThreePhase] = 1 })
		require.NoError(t, err)
		assert.Equal(t, 3, wb.phases)
	})
}

func TestPolichargerStatus(t *testing.T) {
	tc := []struct {
		name   string
		mod    func(map[uint16]uint16)
		status api.ChargeStatus
	}{
		{"plugged", nil, api.StatusB},
		{"unplugged", func(r map[uint16]uint16) { r[polRegPinPlugged] = 0 }, api.StatusA},
		{"plugged, normal connected flag", func(r map[uint16]uint16) {
			r[polRegPinPlugged] = 0
			r[polRegConnected] = 1
		}, api.StatusB},
		{"charging", func(r map[uint16]uint16) { r[polRegCharging] = 1 }, api.StatusC},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			wb, h, err := polCharger(t, tc.mod)
			require.NoError(t, err)

			status, err := wb.Status()
			require.NoError(t, err)
			assert.Equal(t, tc.status, status)
			assert.Empty(t, h.written())
		})
	}
}

func TestPolichargerEnable(t *testing.T) {
	tc := []struct {
		name    string
		mod     func(map[uint16]uint16)
		enable  bool
		current uint16
		writes  []polWrite
	}{
		{"enable armed", nil, true, 6, []polWrite{{polRegLock, 0}}},
		{"enable sets the cap first", nil, true, 10, []polWrite{{polRegLimit, 10}, {polRegLock, 0}}},
		{"enable after the car ended a session", func(r map[uint16]uint16) { r[polRegSetpoint] = 8 }, true, 8,
			[]polWrite{{polRegLimit, 8}, {polRegSetpoint, polArmCurrent}, {polRegLock, 0}}},
		{"enable while charging", func(r map[uint16]uint16) { r[polRegCharging] = 1 }, true, 6, []polWrite{{polRegLock, 0}}},
		{"disable rearms", func(r map[uint16]uint16) {
			r[polRegLock] = 0
			r[polRegSetpoint] = 12
		}, false, 12, []polWrite{{polRegLock, 1}, {polRegLimit, 12}, {polRegSetpoint, polArmCurrent}}},
		{"disable while stopping", func(r map[uint16]uint16) {
			r[polRegLock] = 0
			r[polRegCharging] = 1
		}, false, 12, []polWrite{{polRegLock, 1}}},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			wb, h, err := polCharger(t, tc.mod)
			require.NoError(t, err)
			h.written()
			wb.current = tc.current

			require.NoError(t, wb.Enable(tc.enable))
			assert.Equal(t, tc.writes, h.written())
		})
	}
}

func TestPolichargerMaxCurrent(t *testing.T) {
	t.Run("idle", func(t *testing.T) {
		wb, h, err := polCharger(t, nil)
		require.NoError(t, err)

		require.NoError(t, wb.MaxCurrent(11))
		assert.Equal(t, []polWrite{{polRegLimit, 11}}, h.written())

		// above 16 A the setpoint must not limit the session once it starts
		require.NoError(t, wb.MaxCurrent(20))
		assert.Equal(t, []polWrite{{polRegLimit, 20}, {polRegSetpoint, 20}}, h.written())

		require.NoError(t, wb.MaxCurrent(10))
		assert.Equal(t, []polWrite{{polRegLimit, 10}, {polRegSetpoint, polArmCurrent}}, h.written())
	})

	t.Run("charging", func(t *testing.T) {
		wb, h, err := polCharger(t, func(r map[uint16]uint16) {
			r[polRegCharging] = 1
			r[polRegLimit] = 12
		})
		require.NoError(t, err)

		require.NoError(t, wb.MaxCurrent(9))
		assert.Equal(t, []polWrite{{polRegSetpoint, 9}}, h.written())

		require.NoError(t, wb.MaxCurrent(14))
		assert.Equal(t, []polWrite{{polRegSetpoint, 14}, {polRegLimit, 14}}, h.written())
	})

	t.Run("range", func(t *testing.T) {
		wb, h, err := polCharger(t, nil)
		require.NoError(t, err)

		for _, c := range []int64{5, 33} {
			assert.Error(t, wb.MaxCurrent(c))
		}
		assert.Empty(t, h.written())
	})
}

func TestPolichargerMeasurements(t *testing.T) {
	tc := []struct {
		name       string
		phases     uint16
		charging   uint16
		current    uint16
		power      float64
		l1, l2, l3 float64
	}{
		{"1p", 0, 1, 13, 2990, 13, 0, 0},
		{"3p", 1, 1, 10, 6900, 10, 10, 10},
		{"residual value while idle", 0, 0, 1, 0, 0, 0, 0},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			wb, _, err := polCharger(t, func(r map[uint16]uint16) {
				r[polRegThreePhase] = tc.phases
				r[polRegCharging] = tc.charging
				r[polRegCurrent] = tc.current
			})
			require.NoError(t, err)

			power, err := wb.CurrentPower()
			require.NoError(t, err)
			assert.Equal(t, tc.power, power)

			l1, l2, l3, err := wb.Currents()
			require.NoError(t, err)
			assert.Equal(t, []float64{tc.l1, tc.l2, tc.l3}, []float64{l1, l2, l3})
		})
	}
}
