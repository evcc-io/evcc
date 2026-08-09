package vehicle

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTeslaFleetConfigDecoding(t *testing.T) {
	var config TeslaFleetConfig
	require.NoError(t, util.DecodeOther(map[string]any{
		"credentials": map[string]any{
			"id":     "client",
			"secret": "secret",
		},
		"tokens": map[string]any{
			"access":  "access",
			"refresh": "refresh",
		},
	}, &config))

	assert.Equal(t, "client", config.Credentials.ID)
	assert.Equal(t, "secret", config.Credentials.Secret)
	assert.Equal(t, "access", config.Tokens.Access)
	assert.Equal(t, "refresh", config.Tokens.Refresh)
}

func TestTeslaFleetConfigValidate(t *testing.T) {
	tests := []struct {
		name     string
		config   TeslaFleetConfig
		want     string
		sentinel error
	}{
		{
			name: "missing client id",
			want: "missing client id",
		},
		{
			name: "missing tokens",
			config: TeslaFleetConfig{
				Credentials: ClientCredentials{ID: "client"},
			},
			sentinel: api.ErrMissingToken,
		},
		{
			name: "missing refresh token",
			config: TeslaFleetConfig{
				Credentials: ClientCredentials{ID: "client"},
				Tokens:      Tokens{Access: "access"},
			},
			sentinel: api.ErrMissingToken,
		},
		{
			name: "valid",
			config: TeslaFleetConfig{
				Credentials: ClientCredentials{ID: "client"},
				Tokens:      Tokens{Access: "access", Refresh: "refresh"},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()
			if tc.sentinel != nil {
				assert.ErrorIs(t, err, tc.sentinel)
				return
			}
			if tc.want != "" {
				assert.ErrorContains(t, err, tc.want)
				return
			}
			assert.NoError(t, err)
		})
	}
}
