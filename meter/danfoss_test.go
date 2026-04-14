package meter

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

// TestDanfossTLXInterfaceCompliance verifies interface compliance at compile
// time. No network or hardware access needed.
func TestDanfossTLXInterfaceCompliance(t *testing.T) {
	var _ api.Meter = (*DanfossTLX)(nil)
}

// TestDanfossTLXConfigRejectsNonPV ensures the factory rejects usage modes
// other than "pv" before touching any I/O.
func TestDanfossTLXConfigRejectsNonPV(t *testing.T) {
	_, err := NewDanfossTLXFromConfig(t.Context(), map[string]any{
		"usage":  "grid",
		"device": "/dev/null",
	})
	assert.ErrorContains(t, err, "pv", "non-pv usage must be rejected")

	_, err = NewDanfossTLXFromConfig(t.Context(), map[string]any{
		"usage":  "battery",
		"device": "/dev/null",
	})
	assert.ErrorContains(t, err, "pv", "non-pv usage must be rejected")
}

// TestDanfossTLXConfigRejectsDeviceAndURI verifies that supplying both device
// and uri is rejected before any I/O.
func TestDanfossTLXConfigRejectsDeviceAndURI(t *testing.T) {
	_, err := NewDanfossTLXFromConfig(t.Context(), map[string]any{
		"usage":  "pv",
		"device": "/dev/null",
		"uri":    "host:4196",
	})
	assert.ErrorContains(t, err, "mutually exclusive")
}

// TestDanfossTLXConfigRejectsNoTransport verifies that omitting both device
// and uri returns a clear error.
func TestDanfossTLXConfigRejectsNoTransport(t *testing.T) {
	_, err := NewDanfossTLXFromConfig(t.Context(), map[string]any{
		"usage": "pv",
	})
	assert.ErrorContains(t, err, "device")
}
