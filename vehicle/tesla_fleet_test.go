package vehicle

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

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
