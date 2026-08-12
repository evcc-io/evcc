package charger

import (
	"encoding/binary"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var _ api.Charger = (*Voltie)(nil)

// voltieBlock builds a block payload from register values
func voltieBlock(regs ...uint16) []byte {
	b := make([]byte, 2*len(regs))
	for i, r := range regs {
		binary.BigEndian.PutUint16(b[2*i:], r)
	}
	return b
}

// voltieU32 splits a 32 bit value into the most-significant-word-first pair used
// by the 0x2000 register block
func voltieU32(v uint32) []uint16 {
	return []uint16{uint16(v >> 16), uint16(v)}
}

// voltieStatusReader returns a reader over a synthetic status block
func voltieStatusReader(state, autoStart, enabled, charging, phases, stopReason, current uint16) *voltieBlockReader {
	return &voltieBlockReader{
		block: modbus.Block{Register: voltieRegStatusBlock, Count: voltieLenStatusBlock},
		payload: voltieBlock(
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
		),
	}
}

// voltieMeterReader returns a reader over a synthetic meter block
func voltieMeterReader(voltage, current, duration, energy, power, capacity uint32, tempMcu, tempPower int16) *voltieBlockReader {
	var regs []uint16
	for range 3 {
		regs = append(regs, voltieU32(voltage)...) // 0x2000 voltages
	}
	for range 3 {
		regs = append(regs, voltieU32(current)...) // 0x2006 currents
	}
	regs = append(regs, voltieU32(duration)...) // 0x200C
	regs = append(regs, voltieU32(energy)...)   // 0x200E
	regs = append(regs, voltieU32(power)...)    // 0x2010
	regs = append(regs, voltieU32(capacity)...) // 0x2012
	regs = append(regs, uint16(tempMcu), uint16(tempPower))

	return &voltieBlockReader{
		block:   modbus.Block{Register: voltieRegMeterBlock, Count: voltieLenMeterBlock},
		payload: voltieBlock(regs...),
	}
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
		{0x04, 0, api.StatusNone, "vehicle state D, charging with ventilation (0x04)"},
		{0x06, 5, api.StatusNone, "GFCI fault (0x06): GFCI sensor tripped"},
		{0x0D, 0, api.StatusNone, "vehicle state E, vehicle error (0x0D)"},
		{0x13, 7, api.StatusNone, "booting (0x13): firmware restart"},
		{0x1F, 99, api.StatusNone, "unknown state (0x1F): stop reason 99"},
	}

	for _, tc := range tc {
		status, ok := voltieChargeStatus(tc.state)
		assert.Equal(t, tc.status, status, "state 0x%02X", tc.state)
		assert.Equal(t, tc.err == "", ok, "state 0x%02X", tc.state)

		if tc.err == "" {
			continue
		}

		r := voltieStatusReader(tc.state, 0, 0, 0, 0, tc.stopReason, 16000)

		err := voltieStateError(r, tc.state)
		require.Error(t, err, "state 0x%02X", tc.state)
		assert.Equal(t, tc.err, err.Error())
	}
}

func TestVoltieStatusDecoding(t *testing.T) {
	// state C, auto start off, enabled, charging, 3 phases, no stop reason, 12.5 A
	r := voltieStatusReader(0x03, 0, 1, 1, 3, 0, 12500)

	assert.Equal(t, uint16(0x03), r.u16(voltieRegStatus))
	assert.Equal(t, uint16(0), r.u16(voltieRegAutoStart))
	assert.Equal(t, uint16(1), r.u16(voltieRegChargeEnable))
	assert.Equal(t, uint16(1), r.u16(voltieRegCharging))
	assert.Equal(t, uint16(3), r.u16(voltieRegPhases))
	assert.Equal(t, uint16(12500), r.u16(voltieRegCurrent))
	require.NoError(t, r.err)
}

func TestVoltieMeterDecoding(t *testing.T) {
	// 230.1 V, 16.05 A, 1h05m, 5 kWh, 11040 W, 16 A capacity
	r := voltieMeterReader(230100, 16050, 3900, 18e6, 11040, 16000, 27, -3)

	assert.Equal(t, uint32(230100), r.u32(voltieRegVoltages))
	assert.Equal(t, uint32(16050), r.u32(voltieRegCurrents))
	assert.Equal(t, uint32(3900), r.u32(voltieRegDuration))
	assert.Equal(t, uint32(18e6), r.u32(voltieRegEnergy))
	assert.Equal(t, int32(11040), r.i32(voltieRegPower))
	assert.Equal(t, uint32(16000), r.u32(voltieRegCapacity))
	assert.Equal(t, int16(27), r.i16(voltieRegTempMcu))
	assert.Equal(t, int16(-3), r.i16(voltieRegTempPower))
	require.NoError(t, r.err)
}

// TestVoltieSerial verifies that serial numbers are decoded least-significant
// word first, unlike the 32 bit metering values in the 0x2000 block
func TestVoltieSerial(t *testing.T) {
	r := &voltieBlockReader{
		block: modbus.Block{Register: voltieRegInfoBlock, Count: voltieLenInfoBlock},
		payload: voltieBlock(
			0, 0, // 0x0000 charger id, 0x0001 firmware
			0x3210, 0x7654, 0xBA98, 0xFEDC, // 0x0002 MCU serial
			0x0001, 0x0000, 0x0000, 0x0000, // 0x0006 power board serial
		),
	}

	assert.Equal(t, uint64(0xFEDCBA9876543210), r.serial(voltieRegMcuSerial))
	assert.Equal(t, uint64(1), r.serial(voltieRegHpowSerial))
	require.NoError(t, r.err)
}

// TestVoltieOutOfBlock verifies that a register outside the block surfaces an
// error instead of silently decoding neighbouring data
func TestVoltieOutOfBlock(t *testing.T) {
	r := voltieStatusReader(0x03, 0, 1, 1, 3, 0, 16000)

	r.u16(voltieRegStatusBlock + voltieLenStatusBlock)
	assert.Error(t, r.err)
}

func TestVoltieMaxCurrentBounds(t *testing.T) {
	wb := new(Voltie)

	assert.Error(t, wb.MaxCurrentMillis(5.9))
	assert.Error(t, wb.MaxCurrentMillis(0))
	assert.Error(t, wb.MaxCurrentMillis(-1))
	assert.Error(t, wb.MaxCurrentMillis(32.1), "above the charger's ampacity")
	assert.Error(t, wb.MaxCurrentMillis(66), "would overflow the INT16 register")
}
