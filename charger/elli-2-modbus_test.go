package charger

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
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
