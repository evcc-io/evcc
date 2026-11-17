package charger

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
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

type elliTestConn struct {
	mu      sync.Mutex
	input   map[uint16]uint16
	holding map[uint16]uint16
	writes  chan uint16
	fail    bool
}

func (c *elliTestConn) read(regs map[uint16]uint16, addr, count uint16) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	b := make([]byte, count*2)
	for i := range count {
		value, ok := regs[addr+i]
		if !ok {
			return nil, errors.New("unknown register")
		}
		binary.BigEndian.PutUint16(b[2*i:], value)
	}
	return b, nil
}

func (c *elliTestConn) ReadInputRegisters(addr, count uint16) ([]byte, error) {
	return c.read(c.input, addr, count)
}

func (c *elliTestConn) ReadHoldingRegisters(addr, count uint16) ([]byte, error) {
	return c.read(c.holding, addr, count)
}

func (c *elliTestConn) WriteSingleRegister(addr, value uint16) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if addr != elliRegCurrent || c.fail {
		return nil, errors.New("write failed")
	}
	c.holding[addr] = value
	if c.writes != nil {
		c.writes <- value
	}
	return nil, nil
}

func elliTestCharger() (*Elli2Modbus, *elliTestConn) {
	conn := &elliTestConn{
		input:   map[uint16]uint16{},
		holding: map[uint16]uint16{elliRegCurrent: 0},
	}
	return &Elli2Modbus{
		conn: conn, log: util.NewLogger("elli-test"),
		min: 6, max: 16, current: 60,
	}, conn
}

func TestElli2ModbusStatus(t *testing.T) {
	wb, conn := elliTestCharger()
	for _, tc := range []struct {
		state  uint16
		status api.ChargeStatus
		err    bool
	}{
		{2, api.StatusA, false}, {3, api.StatusA, false},
		{4, api.StatusB, false}, {5, api.StatusB, false}, {6, api.StatusB, false},
		{7, api.StatusC, false}, {8, api.StatusC, false},
		{9, api.StatusNone, true}, {10, api.StatusNone, true}, {11, api.StatusNone, true},
		{0, api.StatusNone, true},
	} {
		conn.input[elliRegStatus] = tc.state
		status, err := wb.Status()
		assert.Equal(t, tc.status, status)
		assert.Equal(t, tc.err, err != nil)
	}
	delete(conn.input, elliRegStatus)
	_, err := wb.Status()
	assert.Error(t, err)
}

func TestElli2ModbusControl(t *testing.T) {
	wb, conn := elliTestCharger()
	require.NoError(t, wb.MaxCurrentMillis(12.3))
	assert.Equal(t, uint16(123), conn.holding[elliRegCurrent])

	// The charger can report a lower effective limit than the command.
	conn.holding[elliRegCurrent] = 60
	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)
	require.NoError(t, wb.Enable(false))
	assert.Equal(t, uint16(0), conn.holding[elliRegCurrent])
	require.NoError(t, wb.Enable(true))
	assert.Equal(t, uint16(123), conn.holding[elliRegCurrent])

	for _, current := range []float64{5.9, 16.1, math.NaN(), math.Inf(1)} {
		assert.Error(t, wb.MaxCurrentMillis(current))
	}
	assert.Equal(t, uint16(123), conn.holding[elliRegCurrent])

	conn.fail = true
	assert.Error(t, wb.MaxCurrentMillis(10))
	assert.Equal(t, uint16(123), wb.command)
	assert.Equal(t, uint16(123), wb.current)
	assert.False(t, wb.written)
}

func TestElli2ModbusMeasurements(t *testing.T) {
	wb, conn := elliTestCharger()
	conn.input = map[uint16]uint16{
		elliRegCurrents: 101, elliRegCurrents + 1: 102, elliRegCurrents + 2: 103,
		elliRegVoltages: 230, elliRegVoltages + 1: 231, elliRegVoltages + 2: 232,
		elliRegPower: 7000, elliRegEnergy: 1, elliRegEnergy + 1: 34464,
	}
	power, err := wb.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 7000.0, power)
	energy, err := wb.TotalEnergy()
	require.NoError(t, err)
	assert.Equal(t, 100.0, energy)
	l1, l2, l3, err := wb.Currents()
	require.NoError(t, err)
	assert.Equal(t, []float64{10.1, 10.2, 10.3}, []float64{l1, l2, l3})
	l1, l2, l3, err = wb.Voltages()
	require.NoError(t, err)
	assert.Equal(t, []float64{230, 231, 232}, []float64{l1, l2, l3})
	delete(conn.input, elliRegPower)
	_, err = wb.CurrentPower()
	assert.Error(t, err)
}

func TestElli2ModbusInitialization(t *testing.T) {
	_, conn := elliTestCharger()
	conn.input[elliRegMax] = 16
	conn.input[elliRegMax+1] = 6
	conn.holding[elliRegTimeout] = 0
	wb, err := newElli2Modbus(t.Context(), conn, util.NewLogger("elli-test"))
	require.NoError(t, err)
	assert.Equal(t, uint16(60), wb.current)

	conn.input[elliRegMax] = 5
	_, err = newElli2Modbus(t.Context(), conn, util.NewLogger("elli-test"))
	assert.ErrorContains(t, err, "invalid current limits")
	delete(conn.input, elliRegMax)
	_, err = newElli2Modbus(t.Context(), conn, util.NewLogger("elli-test"))
	assert.ErrorContains(t, err, "current limits")
	conn.input[elliRegMax] = 16
	delete(conn.holding, elliRegTimeout)
	_, err = newElli2Modbus(t.Context(), conn, util.NewLogger("elli-test"))
	assert.ErrorContains(t, err, "watchdog timeout")
}

func TestElli2ModbusHeartbeat(t *testing.T) {
	wb, conn := elliTestCharger()
	conn.writes = make(chan uint16, 10)
	ctx, cancel := context.WithCancel(t.Context())
	done := make(chan struct{})
	go func() {
		wb.heartbeat(ctx, 10*time.Millisecond)
		close(done)
	}()

	require.NoError(t, wb.MaxCurrent(10))
	assert.Equal(t, uint16(100), <-conn.writes)
	select {
	case value := <-conn.writes:
		assert.Equal(t, uint16(100), value)
	case <-time.After(time.Second):
		t.Fatal("heartbeat did not write")
	}
	cancel()
	<-done
}

type elliWireHandler struct {
	mbserver.RequestHandler
	writes chan uint16
}

func (h *elliWireHandler) HandleInputRegisters(req *mbserver.InputRegistersRequest) ([]uint16, error) {
	if req.UnitId != 1 {
		return nil, mbserver.ErrIllegalDataAddress
	}
	switch {
	case req.Addr == 100 && req.Quantity == 2:
		return []uint16{20, 8}, nil
	case req.Addr == 5 && req.Quantity == 1:
		return []uint16{7}, nil
	default:
		return nil, mbserver.ErrIllegalDataAddress
	}
}

func (h *elliWireHandler) HandleHoldingRegisters(req *mbserver.HoldingRegistersRequest) ([]uint16, error) {
	if req.UnitId != 1 {
		return nil, mbserver.ErrIllegalDataAddress
	}
	switch {
	case !req.IsWrite && req.Addr == 257 && req.Quantity == 1:
		return []uint16{0}, nil
	case req.IsWrite && req.WriteFuncCode == 6 && req.Addr == 261 && len(req.Args) == 1:
		h.writes <- req.Args[0]
		return req.Args, nil
	default:
		return nil, mbserver.ErrIllegalDataAddress
	}
}

func TestElli2ModbusWire(t *testing.T) {
	handler := &elliWireHandler{
		RequestHandler: new(mbserver.DummyHandler),
		writes:         make(chan uint16, 2),
	}
	listener, err := net.Listen("tcp", "localhost:0")
	require.NoError(t, err)
	server, err := mbserver.New(handler)
	require.NoError(t, err)
	// Stop races the server's accept goroutine, as in the other Modbus tests.
	require.NoError(t, server.Start(listener))

	conn, err := modbus.NewConnection(t.Context(), listener.Addr().String(), "", "", 0, modbus.Tcp, 1)
	require.NoError(t, err)
	wb, err := newElli2Modbus(t.Context(), conn, util.NewLogger("elli-test"))
	require.NoError(t, err)
	minCurrent, maxCurrent, err := wb.GetMinMaxCurrent()
	require.NoError(t, err)
	assert.Equal(t, 8.0, minCurrent)
	assert.Equal(t, 16.0, maxCurrent)
	status, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)
	require.NoError(t, wb.MaxCurrentMillis(12.3))
	assert.Equal(t, uint16(123), <-handler.writes)
	require.NoError(t, wb.Enable(false))
	assert.Equal(t, uint16(0), <-handler.writes)
}
