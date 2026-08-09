package charger

import (
	"encoding/binary"
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ api.Charger = (*Voltie)(nil)

// voltieBlock builds a register block from register values
func voltieBlock(regs ...uint16) []byte {
	b := make([]byte, 2*len(regs))
	for i, r := range regs {
		binary.BigEndian.PutUint16(b[2*i:], r)
	}
	return b
}

// voltieTestCharger returns a charger backed by static register blocks
func voltieTestCharger(status, meter []byte) *Voltie {
	wb := new(Voltie)
	wb.log = util.NewLogger("voltie")
	wb.statusG = util.ResettableCached(func() ([]byte, error) {
		return status, nil
	}, time.Minute)
	wb.meterG = util.ResettableCached(func() ([]byte, error) {
		return meter, nil
	}, time.Minute)

	return wb
}

// voltieStatusBlock returns a status block, 0x000A..0x0015
func voltieStatusBlock(state, autoStart, enabled, charging, phases, stopReason, current uint16) []byte {
	return voltieBlock(
		state,      // 0x000A status
		autoStart,  // 0x000B auto start
		enabled,    // 0x000C charging enabled
		charging,   // 0x000D charging
		phases,     // 0x000E phases
		0,          // 0x000F set DLM
		0,          // 0x0010 config flags
		0,          // 0x0011 mode flags
		stopReason, // 0x0012 stop reason
		0,          // 0x0013 autonomous current limit
		current,    // 0x0014 software current limit
		0,          // 0x0015 effective DLM
	)
}

// voltieMeterBlock returns a meter block, 0x2000..0x2015
func voltieMeterBlock(voltage, current, duration, energy, power, capacity uint32, tempMcu, tempPower int16) []byte {
	u32 := func(v uint32) (uint16, uint16) {
		return uint16(v >> 16), uint16(v)
	}

	uHi, uLo := u32(voltage)
	iHi, iLo := u32(current)
	dHi, dLo := u32(duration)
	eHi, eLo := u32(energy)
	pHi, pLo := u32(power)
	cHi, cLo := u32(capacity)

	return voltieBlock(
		uHi, uLo, uHi, uLo, uHi, uLo, // 0x2000 voltages
		iHi, iLo, iHi, iLo, iHi, iLo, // 0x2006 currents
		dHi, dLo, // 0x200C duration
		eHi, eLo, // 0x200E energy
		pHi, pLo, // 0x2010 power
		cHi, cLo, // 0x2012 current capacity
		uint16(tempMcu),   // 0x2014
		uint16(tempPower), // 0x2015
	)
}

func TestVoltieStatus(t *testing.T) {
	tc := []struct {
		state      uint16
		stopReason uint16
		status     api.ChargeStatus
		err        string
	}{
		{0x01, 10, api.StatusA, ""},
		{0x02, 4, api.StatusB, ""},
		{0x03, 0, api.StatusC, ""},
		{0x04, 0, api.StatusC, ""},
		{0x06, 5, api.StatusNone, "GFCI fault (0x06): GFCI sensor tripped"},
		{0x0D, 0, api.StatusNone, "vehicle state E, vehicle error (0x0D)"},
		{0x13, 7, api.StatusNone, "booting (0x13): firmware restart"},
		{0x1F, 99, api.StatusNone, "unknown state (0x1F): stop reason 99"},
	}

	for _, tc := range tc {
		wb := voltieTestCharger(voltieStatusBlock(tc.state, 0, 0, 0, 0, tc.stopReason, 16000), nil)

		status, err := wb.Status()
		assert.Equal(t, tc.status, status, "state 0x%02X", tc.state)

		if tc.err == "" {
			assert.NoError(t, err, "state 0x%02X", tc.state)
		} else {
			require.Error(t, err, "state 0x%02X", tc.state)
			assert.Equal(t, tc.err, err.Error())
		}
	}
}

func TestVoltieStatusBlockDecoding(t *testing.T) {
	// state C, auto start off, enabled, charging, 3 phases, no stop reason, 12.5 A
	wb := voltieTestCharger(voltieStatusBlock(0x03, 0, 1, 1, 3, 0, 12500), nil)

	enabled, err := wb.Enabled()
	require.NoError(t, err)
	assert.True(t, enabled)

	current, err := wb.GetMaxCurrent()
	require.NoError(t, err)
	assert.Equal(t, 12.5, current)

	phases, err := wb.GetPhases()
	require.NoError(t, err)
	assert.Equal(t, 3, phases)
}

func TestVoltieMeterBlockDecoding(t *testing.T) {
	// 230.1 V, 16.05 A, 1h05m, 5 kWh, 11040 W, 16 A capacity
	wb := voltieTestCharger(nil, voltieMeterBlock(230100, 16050, 3900, 18e6, 11040, 16000, 27, -3))

	power, err := wb.CurrentPower()
	require.NoError(t, err)
	assert.Equal(t, 11040.0, power)

	energy, err := wb.ChargedEnergy()
	require.NoError(t, err)
	assert.Equal(t, 5.0, energy)

	duration, err := wb.ChargeDuration()
	require.NoError(t, err)
	assert.Equal(t, 65*time.Minute, duration)

	l1, l2, l3, err := wb.Currents()
	require.NoError(t, err)
	assert.Equal(t, []float64{16.05, 16.05, 16.05}, []float64{l1, l2, l3})

	u1, u2, u3, err := wb.Voltages()
	require.NoError(t, err)
	assert.Equal(t, []float64{230.1, 230.1, 230.1}, []float64{u1, u2, u3})
}

// TestVoltieSerial verifies that serial numbers are decoded least-significant
// word first, unlike the metering values in the 0x2000 block
func TestVoltieSerial(t *testing.T) {
	b := voltieBlock(
		0, 0, // 0x0000 charger id, 0x0001 firmware
		0x3210, 0x7654, 0xBA98, 0xFEDC, // 0x0002 MCU serial
		0x0001, 0x0000, 0x0000, 0x0000, // 0x0006 power board serial
	)

	assert.Equal(t, uint64(0xFEDCBA9876543210), voltieSerial(b, voltieOffMcuSerial))
	assert.Equal(t, uint64(1), voltieSerial(b, voltieOffHpowSerial))
}

func TestVoltieMaxCurrentBounds(t *testing.T) {
	wb := voltieTestCharger(nil, nil)

	assert.Error(t, wb.MaxCurrentMillis(5.9))
	assert.Error(t, wb.MaxCurrentMillis(0))
	assert.Error(t, wb.MaxCurrentMillis(-1))
	assert.Error(t, wb.MaxCurrentMillis(66))
}
