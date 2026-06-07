package plugin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestAA55UDPFromConfig_Block verifies that the nested block map decodes and a
// target register that fits within the block is accepted (block-read mode).
func TestAA55UDPFromConfig_Block(t *testing.T) {
	_, err := NewAA55UDPFromConfig(context.Background(), map[string]any{
		"host":     "127.0.0.1",
		"id":       247,
		"register": 35139,
		"count":    2,
		"block":    map[string]any{"register": 35100, "count": 125},
		"decode":   "int32be",
	})
	require.NoError(t, err)
}

// TestAA55UDPFromConfig_BlockRejectsOutOfRange verifies that a target register
// outside the configured block is rejected.
func TestAA55UDPFromConfig_BlockRejectsOutOfRange(t *testing.T) {
	_, err := NewAA55UDPFromConfig(context.Background(), map[string]any{
		"host":     "127.0.0.1",
		"id":       247,
		"register": 36017, // outside READ 125 @ 35100
		"count":    2,
		"block":    map[string]any{"register": 35100, "count": 125},
		"decode":   "float32be",
	})
	require.Error(t, err)
}

// TestAA55UDPFromConfig_Register verifies register-read mode (no block).
func TestAA55UDPFromConfig_Register(t *testing.T) {
	_, err := NewAA55UDPFromConfig(context.Background(), map[string]any{
		"host":     "127.0.0.1",
		"register": 30127,
		"count":    2,
		"decode":   "int32be",
	})
	require.NoError(t, err)
}
