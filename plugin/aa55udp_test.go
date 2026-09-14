package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAA55UDPFromConfig_Block verifies the modbus.Register and nested block map
// decode, with the count (2 registers) derived from the int32 decode width.
func TestAA55UDPFromConfig_Block(t *testing.T) {
	_, err := NewAA55UDPFromConfig(context.Background(), map[string]any{
		"host":     "127.0.0.1",
		"id":       247,
		"register": map[string]any{"address": 35139, "decode": "int32"},
		"block":    map[string]any{"register": 35100, "count": 125},
	})
	require.NoError(t, err)
}

// TestAA55UDPFromConfig_BlockRejectsOutOfRange verifies that a target register
// outside the configured block is rejected.
func TestAA55UDPFromConfig_BlockRejectsOutOfRange(t *testing.T) {
	_, err := NewAA55UDPFromConfig(context.Background(), map[string]any{
		"host":     "127.0.0.1",
		"id":       247,
		"register": map[string]any{"address": 36017, "decode": "float32"}, // outside READ 125 @ 35100
		"block":    map[string]any{"register": 35100, "count": 125},
	})
	require.Error(t, err)
}

// TestAA55UDPFromConfig_Register verifies register-read mode (no block).
func TestAA55UDPFromConfig_Register(t *testing.T) {
	_, err := NewAA55UDPFromConfig(context.Background(), map[string]any{
		"host":     "127.0.0.1",
		"register": map[string]any{"address": 30127, "decode": "int32"},
	})
	require.NoError(t, err)
}
