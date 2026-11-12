package charger

import (
	"context"
	"encoding/binary"
	"io"
	"net"
	"slices"
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
	reads  int
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

	h.reads++

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

func (h *polHandler) written() []polWrite {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]polWrite(nil), h.writes...)
}

func (h *polHandler) reg(addr uint16) uint16 {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.regs[addr]
}

func (h *polHandler) readCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.reads
}

func (h *polHandler) reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.writes = nil
	h.reads = 0
}

// polRegsIdle returns a locked, armed, single-phase charger with a car connected
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

// polMux dispatches requests to one mock charger per slave id, so tests share a
// server: mbserver.Stop() races its accept goroutine and is never called
type polMux struct {
	mbserver.RequestHandler
	mu       sync.Mutex
	handlers map[uint8]*polHandler
	next     uint8
}

func (m *polMux) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	m.mu.Lock()
	h, ok := m.handlers[req.UnitId]
	m.mu.Unlock()

	if !ok {
		return nil, mbserver.ErrGWTargetFailedToRespond
	}

	return h.HandleHoldingRegisters(req)
}

var (
	polOnce sync.Once
	polURI  string
	polSrv  = &polMux{RequestHandler: new(mbserver.DummyHandler), handlers: make(map[uint8]*polHandler)}
)

// polServer adds a mock charger to the shared server and returns its address and slave id
func polServer(t *testing.T, regs map[uint16]uint16) (string, uint8, *polHandler) {
	t.Helper()

	polOnce.Do(func() {
		l, err := net.Listen("tcp", "localhost:0")
		require.NoError(t, err)

		srv, err := mbserver.New(polSrv)
		require.NoError(t, err)
		require.NoError(t, srv.Start(l))

		polURI = l.Addr().String()
	})

	h := &polHandler{RequestHandler: new(mbserver.DummyHandler), regs: regs}

	polSrv.mu.Lock()
	defer polSrv.mu.Unlock()

	polSrv.next++
	require.Less(t, polSrv.next, uint8(248), "out of slave ids")
	polSrv.handlers[polSrv.next] = h

	return polURI, polSrv.next, h
}

// polTestCharger connects a charger to a new mock charger
func polTestCharger(t *testing.T, regs map[uint16]uint16) (*Policharger, *polHandler) {
	t.Helper()

	uri, id, h := polServer(t, regs)

	conn, err := modbus.NewConnection(t.Context(), uri, "", "", 0, modbus.Tcp, id)
	require.NoError(t, err)

	// no read cache: every call sees the current mock registers
	return newPolicharger(conn, util.NewLogger("test"), 0), h
}

// polProxy forwards Modbus TCP and swallows a number of replies, like a lossy gateway
type polProxy struct {
	mu   sync.Mutex
	drop int
}

func (p *polProxy) dropNext(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.drop = n
}

func (p *polProxy) swallow() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.drop > 0 {
		p.drop--
		return true
	}
	return false
}

func (p *polProxy) serve(c net.Conn, upstream string) {
	defer c.Close()

	u, err := net.Dial("tcp", upstream)
	if err != nil {
		return
	}

	go func() {
		_, _ = io.Copy(u, c)
		u.Close()
	}()

	hdr := make([]byte, 7)
	for {
		if _, err := io.ReadFull(u, hdr); err != nil {
			return
		}

		body := make([]byte, int(binary.BigEndian.Uint16(hdr[4:6]))-1)
		if _, err := io.ReadFull(u, body); err != nil {
			return
		}

		if !p.swallow() {
			if _, err := c.Write(slices.Concat(hdr, body)); err != nil {
				return
			}
		}
	}
}

// polLossyCharger connects a charger to a mock charger through a lossy proxy
func polLossyCharger(t *testing.T, regs map[uint16]uint16) (*Policharger, *polHandler, *polProxy) {
	t.Helper()

	upstream, id, h := polServer(t, regs)

	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	t.Cleanup(func() { l.Close() })

	p := new(polProxy)
	go func() {
		for {
			c, err := l.Accept()
			if err != nil {
				return
			}
			go p.serve(c, upstream)
		}
	}()

	conn, err := modbus.NewConnection(t.Context(), l.Addr().String(), "", "", 0, modbus.Tcp, id)
	require.NoError(t, err)
	conn.Timeout(200 * time.Millisecond)

	return newPolicharger(conn, util.NewLogger("test"), 0), h, p
}

func TestPolichargerNew(t *testing.T) {
	regs := polRegsIdle()
	regs[polRegLockType] = 0 // factory default: normal mode
	regs[polRegLock] = 0
	regs[polRegThreePhase] = 1
	uri, id, h := polServer(t, regs)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	c, err := NewPolichargerFromConfig(ctx, map[string]any{"uri": uri, "id": id})
	require.NoError(t, err)

	wb := c.(*Policharger)
	assert.Equal(t, 3, wb.phases)

	// locked before switching to normally locked mode
	w := h.written()
	require.GreaterOrEqual(t, len(w), 2)
	assert.Equal(t, []polWrite{{polRegLock, 1}, {polRegLockType, 1}}, w[:2])
	assert.Equal(t, uint16(1), h.reg(polRegLockType))
}

func TestPolichargerNewLocksIdle(t *testing.T) {
	tc := []struct {
		name     string
		charging uint16
		writes   []polWrite
	}{
		{"idle and unlocked: lock", 0, []polWrite{{polRegLock, 1}}},
		{"charging: leave the session alone", 1, nil},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			regs := polRegsIdle()
			regs[polRegLock] = 0
			regs[polRegCharging] = tc.charging
			uri, id, h := polServer(t, regs)

			ctx, cancel := context.WithCancel(t.Context())
			defer cancel()

			_, err := NewPolichargerFromConfig(ctx, map[string]any{"uri": uri, "id": id})
			require.NoError(t, err)

			if tc.writes == nil {
				assert.NotContains(t, h.written(), polWrite{polRegLock, 1})
			} else {
				assert.Equal(t, tc.writes, h.written()[:len(tc.writes)])
			}
		})
	}
}

func TestPolichargerNewPvLimitUnreadable(t *testing.T) {
	regs := polRegsIdle()
	delete(regs, polRegPvGridLimit) // the read returns an exception
	uri, id, _ := polServer(t, regs)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	_, err := NewPolichargerFromConfig(ctx, map[string]any{"uri": uri, "id": id})
	assert.NoError(t, err, "the PV grid limit is advisory only")
}

func TestPolichargerNewUnreachable(t *testing.T) {
	l, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	uri := l.Addr().String()
	l.Close()

	_, err = NewPolicharger(t.Context(), modbus.Settings{URI: uri, ID: 1, Timeout: 100 * time.Millisecond})
	assert.Error(t, err)
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
		{"not armed, normal mode: status does not write", func(r map[uint16]uint16) {
			r[polRegLockType] = 0
			r[polRegSetpoint] = 0
		}, api.StatusB},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			regs := polRegsIdle()
			tc.mod(regs)
			wb, h := polTestCharger(t, regs)

			status, err := wb.Status()
			require.NoError(t, err)
			assert.Equal(t, tc.status, status)
			assert.Empty(t, h.written(), "status must be read-only")
		})
	}
}

func TestPolichargerMaintain(t *testing.T) {
	tc := []struct {
		name    string
		mod     func(map[uint16]uint16)
		current uint16
		cfgAge  time.Duration // since the last lock mode check, 0 = never checked
		writes  []polWrite
	}{
		{
			name: "armed: nothing to do",
			mod:  func(r map[uint16]uint16) {}, current: 6,
		},
		{
			name: "car ended the session below 16 A: cap first, then arm",
			mod: func(r map[uint16]uint16) {
				r[polRegSetpoint] = 9
				r[polRegManualLimit] = 16
			},
			current: 9,
			writes:  []polWrite{{polRegManualLimit, 9}, {polRegSetpoint, polArmCurrent}},
		},
		{
			name: "charging: never arm",
			mod: func(r map[uint16]uint16) {
				r[polRegLock] = 0
				r[polRegCharging] = 1
				r[polRegSetpoint] = 10
				r[polRegManualLimit] = 10
			},
			current: 10,
		},
		{
			name: "back in normal mode: lock and restore mode before arming",
			mod: func(r map[uint16]uint16) {
				r[polRegLockType] = 0
				r[polRegLock] = 0
				r[polRegSetpoint] = 0
			},
			current: 6,
			writes:  []polWrite{{polRegLock, 1}, {polRegLockType, 1}, {polRegSetpoint, polArmCurrent}},
		},
		{
			name: "periodic check not due: the check before arming catches normal mode",
			mod: func(r map[uint16]uint16) {
				r[polRegLockType] = 0
				r[polRegLock] = 0
				r[polRegSetpoint] = 0
			},
			current: 6,
			cfgAge:  2 * time.Minute,
			writes:  []polWrite{{polRegLock, 1}, {polRegLockType, 1}, {polRegSetpoint, polArmCurrent}},
		},
		{
			name: "already armed but back in normal mode: lock and restore mode",
			mod: func(r map[uint16]uint16) {
				r[polRegLockType] = 0
				r[polRegLock] = 0
			},
			current: 6,
			cfgAge:  2 * time.Minute,
			writes:  []polWrite{{polRegLock, 1}, {polRegLockType, 1}},
		},
		{
			name: "stale cap after a failed write: corrected",
			mod: func(r map[uint16]uint16) {
				r[polRegManualLimit] = 12
			},
			current: 8,
			writes:  []polWrite{{polRegManualLimit, 8}},
		},
		{
			name: "cap above 16 A: setpoint follows the cap",
			mod: func(r map[uint16]uint16) {
				r[polRegManualLimit] = 6
			},
			current: 20,
			writes:  []polWrite{{polRegManualLimit, 20}, {polRegSetpoint, 20}},
		},
		{
			name: "low PV grid limit: warn, do not change it",
			mod: func(r map[uint16]uint16) {
				r[polRegPvGridLimit] = 0
			},
			current: 6,
		},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			regs := polRegsIdle()
			tc.mod(regs)
			wb, h := polTestCharger(t, regs)
			wb.current = tc.current
			if tc.cfgAge > 0 {
				wb.cfgOK = time.Now().Add(-tc.cfgAge)
			}

			require.NoError(t, wb.maintain())
			assert.Equal(t, tc.writes, h.written())

			// idempotent
			h.reset()
			require.NoError(t, wb.maintain())
			assert.Empty(t, h.written())
		})
	}
}

func TestPolichargerEnable(t *testing.T) {
	wb, h := polTestCharger(t, polRegsIdle())
	wb.current = 10

	require.NoError(t, wb.Enable(true))
	assert.Equal(t, []polWrite{{polRegManualLimit, 10}, {polRegLock, 0}}, h.written())

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	// again: nothing to do
	h.reset()
	require.NoError(t, wb.Enable(true))
	assert.Empty(t, h.written())
}

func TestPolichargerEnableWhileCharging(t *testing.T) {
	regs := polRegsIdle()
	regs[polRegCharging] = 1 // locked but still ramping down
	regs[polRegSetpoint] = 10
	regs[polRegManualLimit] = 10
	wb, h := polTestCharger(t, regs)
	wb.current = 10

	require.NoError(t, wb.Enable(true))
	assert.Equal(t, []polWrite{{polRegLock, 0}}, h.written(), "no arming while charging")
}

func TestPolichargerDisable(t *testing.T) {
	regs := polRegsIdle()
	regs[polRegLock] = 0
	regs[polRegSetpoint] = 12
	regs[polRegManualLimit] = 12
	wb, h := polTestCharger(t, regs)
	wb.current = 12

	require.NoError(t, wb.Enable(false))
	assert.Equal(t, []polWrite{{polRegLock, 1}, {polRegSetpoint, polArmCurrent}}, h.written(), "lock first, then arm")

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestPolichargerDisableStillCharging(t *testing.T) {
	regs := polRegsIdle()
	regs[polRegLock] = 0
	regs[polRegCharging] = 1 // the mock does not stop by itself
	regs[polRegSetpoint] = 12
	regs[polRegManualLimit] = 12
	wb, h := polTestCharger(t, regs)
	wb.current = 12

	require.NoError(t, wb.Enable(false))
	assert.Equal(t, []polWrite{{polRegLock, 1}}, h.written(), "no arming before the session has ended")
}

func TestPolichargerMaxCurrent(t *testing.T) {
	t.Run("charging", func(t *testing.T) {
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
	})

	t.Run("idle", func(t *testing.T) {
		wb, h := polTestCharger(t, polRegsIdle())

		// only the cap changes, reg 80 stays armed
		require.NoError(t, wb.MaxCurrent(11))
		assert.Equal(t, []polWrite{{polRegManualLimit, 11}}, h.written())

		// above 16 A the setpoint must not limit the session once it starts
		h.reset()
		require.NoError(t, wb.MaxCurrent(20))
		assert.Equal(t, []polWrite{{polRegManualLimit, 20}, {polRegSetpoint, 20}}, h.written())

		// back below 16 A: armed at 16 A again
		h.reset()
		require.NoError(t, wb.MaxCurrent(10))
		assert.Equal(t, []polWrite{{polRegManualLimit, 10}, {polRegSetpoint, polArmCurrent}}, h.written())
	})

	t.Run("range", func(t *testing.T) {
		wb, h := polTestCharger(t, polRegsIdle())

		for _, c := range []int64{0, 5, 33} {
			assert.Error(t, wb.MaxCurrent(c), c)
		}
		assert.Empty(t, h.written())
	})
}

func TestPolichargerMeasurements(t *testing.T) {
	tc := []struct {
		name       string
		phases     int
		charging   uint16
		current    uint16
		power      float64
		l1, l2, l3 float64
	}{
		{"1p charging", 1, 1, 13, 2990, 13, 0, 0},
		{"3p charging", 3, 1, 10, 6900, 10, 10, 10},
		{"residual sensor value while not charging", 1, 0, 1, 0, 0, 0, 0},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			regs := polRegsIdle()
			regs[polRegCharging] = tc.charging
			regs[polRegCurrent] = tc.current
			wb, _ := polTestCharger(t, regs)
			wb.phases = tc.phases

			power, err := wb.CurrentPower()
			require.NoError(t, err)
			assert.Equal(t, tc.power, power)

			l1, l2, l3, err := wb.Currents()
			require.NoError(t, err)
			assert.Equal(t, []float64{tc.l1, tc.l2, tc.l3}, []float64{l1, l2, l3})
		})
	}
}

func TestPolichargerLossyLink(t *testing.T) {
	t.Run("read retried after a lost reply", func(t *testing.T) {
		wb, _, p := polLossyCharger(t, polRegsIdle())

		p.dropNext(1)
		status, err := wb.Status()
		require.NoError(t, err)
		assert.Equal(t, api.StatusB, status)
	})

	t.Run("write applied but echo lost: read back, no second write", func(t *testing.T) {
		wb, h, p := polLossyCharger(t, polRegsIdle())

		p.dropNext(1)
		require.NoError(t, wb.write(polRegManualLimit, 10))
		assert.Equal(t, []polWrite{{polRegManualLimit, 10}}, h.written())
		assert.Equal(t, 1, h.readCount(), "value read back")
	})

	t.Run("gives up after the retries", func(t *testing.T) {
		wb, h, p := polLossyCharger(t, polRegsIdle())

		p.dropNext(2 * polRetries)
		_, err := wb.Status()
		require.Error(t, err)
		assert.Equal(t, polRetries, h.readCount())
	})
}

func TestPolichargerExceptionNotRetried(t *testing.T) {
	wb, h := polTestCharger(t, polRegsIdle())

	_, err := wb.read(200, 1) // outside the register map
	require.Error(t, err)
	assert.True(t, modbus.IsException(err))
	assert.Equal(t, 1, h.readCount())
}

func TestPolichargerDiagnose(t *testing.T) {
	wb, h := polTestCharger(t, polRegsIdle())

	wb.Diagnose()
	assert.Empty(t, h.written(), "diagnose must be read-only")
}
