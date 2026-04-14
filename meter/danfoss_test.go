package meter

import (
	"testing"

	comlynx "github.com/PanterSoft/comlynx-go"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestParseComlynxNodeAddress(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    comlynx.Address
		wantErr string
	}{
		{"valid", "c-6-b1", comlynx.NewAddress(0x0c, 0x06, 0xb1), ""},
		{"valid lowercase", "a-f-ff", comlynx.NewAddress(0x0a, 0x0f, 0xff), ""},
		{"valid uppercase", "A-F-FF", comlynx.NewAddress(0x0a, 0x0f, 0xff), ""},
		{"valid zeros", "0-0-00", comlynx.NewAddress(0x00, 0x00, 0x00), ""},
		{"network out of range", "10-0-00", comlynx.Address{}, "network component"},
		{"subnet out of range", "0-10-00", comlynx.Address{}, "subnet component"},
		{"node out of range", "0-0-100", comlynx.Address{}, "node component"},
		{"invalid format missing parts", "c-6", comlynx.Address{}, "expected format"},
		{"invalid format no dashes", "c6b1", comlynx.Address{}, "expected format"},
		{"invalid hex chars", "g-0-00", comlynx.Address{}, "expected format"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := parseComlynxNodeAddress(tc.input)
			if tc.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.wantErr)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tc.want, got)
		})
	}
}
