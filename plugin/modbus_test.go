package plugin

import (
	"encoding/binary"
	"testing"

	"github.com/evcc-io/evcc/util/modbus"
	gridx "github.com/grid-x/modbus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// huaweiGridBlock mirrors the shared grid block of the huawei-sun2000-hybrid
// template: 16 registers (32 bytes) starting at 37107.
var huaweiGridBlock = modbus.Block{Register: 37107, Count: 16}

func op(addr, length uint16) modbus.RegisterOperation {
	return modbus.RegisterOperation{FuncCode: gridx.FuncCodeReadHoldingRegisters, Addr: addr, Length: length}
}

// TestExtractBlock verifies that each Huawei grid register is sliced from the
// block payload at the correct offset and decodes to the expected value.
func TestExtractBlock(t *testing.T) {
	payload := make([]byte, 2*huaweiGridBlock.Count)
	// power @ 37113 (offset 12), int32 = 1234 W
	binary.BigEndian.PutUint32(payload[12:], uint32(int32(1234)))
	// imported energy @ 37121 (offset 28), uint32 = 567890
	binary.BigEndian.PutUint32(payload[28:], 567890)

	power, err := huaweiGridBlock.Extract(op(37113, 2), payload)
	require.NoError(t, err)
	powerDec, err := (modbus.Register{Type: "holding", Decode: "int32"}).DecodeFunc()
	require.NoError(t, err)
	assert.Equal(t, 1234.0, powerDec(power))

	energy, err := huaweiGridBlock.Extract(op(37121, 2), payload)
	require.NoError(t, err)
	energyDec, err := (modbus.Register{Type: "holding", Decode: "uint32"}).DecodeFunc()
	require.NoError(t, err)
	assert.Equal(t, 567890.0, energyDec(energy))
}

func TestExtractBlockBounds(t *testing.T) {
	payload := make([]byte, 2*huaweiGridBlock.Count)

	// register outside the block is rejected
	_, err := huaweiGridBlock.Extract(op(37200, 2), payload)
	require.Error(t, err)

	// short payload is rejected
	_, err = huaweiGridBlock.Extract(op(37121, 2), payload[:10])
	require.Error(t, err)
}
