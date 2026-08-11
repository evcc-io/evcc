package meter

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPowerWallFromConfigValidation(t *testing.T) {
	tests := []struct {
		name   string
		config map[string]any
		want   string
	}{
		{
			name:   "missing usage",
			config: map[string]any{"password": "secret"},
			want:   "missing usage",
		},
		{
			name:   "missing password",
			config: map[string]any{"usage": "battery"},
			want:   "missing password",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPowerWallFromConfig(t.Context(), tc.config)
			assert.ErrorContains(t, err, tc.want)
		})
	}
}

// decodePowerWallConfig mirrors the decoding done by the meter constructors
func decodePowerWallConfig(t *testing.T, other map[string]any) powerWallConfig {
	t.Helper()

	cc := defaultPowerWallConfig()
	require.NoError(t, util.DecodeOther(other, &cc))
	require.NoError(t, cc.validate())

	return cc
}

func TestPowerWallConfigLegacyUsage(t *testing.T) {
	tests := []struct {
		usage string
		want  string
	}{
		{usage: "grid", want: "site"},
		{usage: "pv", want: "solar"},
	}

	for _, tc := range tests {
		t.Run(tc.usage, func(t *testing.T) {
			cc := decodePowerWallConfig(t, map[string]any{
				"usage":    tc.usage,
				"password": "secret",
			})
			assert.Equal(t, tc.want, cc.Usage)
		})
	}
}

func TestPowerWallConfigDeprecatedParams(t *testing.T) {
	decodePowerWallConfig(t, map[string]any{
		"usage": "battery", "password": "secret",
		"refreshToken": "token", "siteId": 123,
	})
}

func TestNewPowerWallFleetFromConfigValidation(t *testing.T) {
	tests := []struct {
		name         string
		config       map[string]any
		want         string
		wantSentinel error
	}{
		{
			name: "missing client id",
			config: map[string]any{
				"usage":    "battery",
				"password": "secret",
			},
			want: "missing client id",
		},
		{
			name: "missing tokens",
			config: map[string]any{
				"usage":    "battery",
				"password": "secret",
				"credentials": map[string]any{
					"id": "client",
				},
			},
			wantSentinel: api.ErrMissingToken,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPowerWallFleetFromConfig(t.Context(), tc.config)
			if tc.wantSentinel != nil {
				assert.ErrorIs(t, err, tc.wantSentinel)
				return
			}
			assert.ErrorContains(t, err, tc.want)
		})
	}
}

func TestTeslaReserveLimit(t *testing.T) {
	tests := []struct {
		name  string
		limit float64
		want  uint64
	}{
		{name: "negative", limit: -1, want: 0},
		{name: "below cap", limit: 79.9, want: 79},
		{name: "at cap", limit: 80, want: 80},
		{name: "unsupported range start", limit: 81, want: 80},
		{name: "unsupported range end", limit: 99.9, want: 80},
		{name: "full reserve", limit: 100, want: 100},
		{name: "above full reserve", limit: 101, want: 100},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, teslaReserveLimit(tc.limit))
		})
	}
}
