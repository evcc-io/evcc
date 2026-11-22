package charger

import (
	"encoding/binary"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type deyeEvseMock struct {
	regs map[uint16]uint16
}

func (m *deyeEvseMock) ReadHoldingRegisters(address, _ uint16) ([]byte, error) {
	return binary.BigEndian.AppendUint16(nil, m.regs[address]), nil
}

func (m *deyeEvseMock) WriteMultipleRegisters(address, _ uint16, value []byte) ([]byte, error) {
	m.regs[address] = binary.BigEndian.Uint16(value)
	return nil, nil
}

func TestDeyeEvseDisableWithoutVehicle(t *testing.T) {
	conn := &deyeEvseMock{regs: map[uint16]uint16{
		deyeEvseRegState:      0x0281,
		deyeEvseRegPowerLimit: 4140,
	}}
	wb := &DeyeEvse{conn: conn, power: 4140}

	require.NoError(t, wb.Enable(false))
	assert.Equal(t, uint16(4140), conn.regs[deyeEvseRegPowerLimit], "limit kept")

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)
}

func TestDeyeEvseArm(t *testing.T) {
	conn := &deyeEvseMock{regs: map[uint16]uint16{
		deyeEvseRegState:      0x0281,
		deyeEvseRegPowerLimit: 0,
	}}
	wb := &DeyeEvse{conn: conn, power: 1380}

	status, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusA, status)
	assert.Equal(t, uint16(1380), conn.regs[deyeEvseRegPowerLimit])
}

func TestDeyeEvseDisableWhileCharging(t *testing.T) {
	conn := &deyeEvseMock{regs: map[uint16]uint16{
		deyeEvseRegState:      0x0220,
		deyeEvseRegPowerLimit: 3680,
	}}
	wb := &DeyeEvse{conn: conn, power: 3680}

	status, err := wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusC, status)

	require.NoError(t, wb.Enable(false))
	assert.Equal(t, uint16(0), conn.regs[deyeEvseRegPowerLimit])

	// charger falls back to available although the vehicle is still plugged in
	conn.regs[deyeEvseRegState] = 0x0281

	for range 3 {
		status, err = wb.Status()
		require.NoError(t, err)
		assert.Equal(t, api.StatusB, status)
		assert.Equal(t, uint16(0), conn.regs[deyeEvseRegPowerLimit], "must not re-arm")
	}

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.False(t, enabled)

	// vehicle unplugged meanwhile, enabling reveals it
	require.NoError(t, wb.Enable(true))

	status, err = wb.Status()
	require.NoError(t, err)
	assert.Equal(t, api.StatusA, status)
	assert.False(t, wb.connected)
}
