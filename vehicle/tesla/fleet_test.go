package tesla

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/stretchr/testify/assert"
)

// Client must reject an incomplete config before opening any connection
func TestFleetClientValidates(t *testing.T) {
	_, err := FleetConfig{}.Client(util.NewLogger("tesla"))
	assert.ErrorContains(t, err, "missing client id")
}

func TestFleetConfigValidate(t *testing.T) {
	tests := []struct {
		name     string
		config   FleetConfig
		want     string
		sentinel error
	}{
		{
			name: "missing client id",
			want: "missing client id",
		},
		{
			name: "missing tokens",
			config: FleetConfig{
				Credentials: oauth.ClientCredentials{ID: "client"},
			},
			sentinel: api.ErrMissingToken,
		},
		{
			name: "missing refresh token",
			config: FleetConfig{
				Credentials: oauth.ClientCredentials{ID: "client"},
				Tokens:      oauth.Tokens{Access: "access"},
			},
			sentinel: api.ErrMissingToken,
		},
		{
			name: "valid",
			config: FleetConfig{
				Credentials: oauth.ClientCredentials{ID: "client"},
				Tokens:      oauth.Tokens{Access: "access", Refresh: "refresh"},
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
