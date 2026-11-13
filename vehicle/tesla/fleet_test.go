package tesla

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/oauth"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func TestFleetBaseUrl(t *testing.T) {
	const (
		na   = "https://fleet-api.prd.na.vn.cloud.tesla.com"
		eu   = "https://fleet-api.prd.eu.vn.cloud.tesla.com"
		auth = "https://fleet-auth.prd.vn.cloud.tesla.com/oauth2/v3/userinfo"
	)

	tests := []struct {
		name   string
		ouCode string
		aud    []string
		want   string
	}{
		{name: "no audience", want: na},
		{name: "eu only", ouCode: "EU", aud: []string{eu, auth}, want: eu},
		{name: "eu without region", aud: []string{eu, auth}, want: eu},
		{name: "eu region first", ouCode: "EU", aud: []string{eu, na, auth}, want: eu},
		{name: "eu region last", ouCode: "EU", aud: []string{na, eu, auth}, want: eu},
		{name: "na region", ouCode: "NA", aud: []string{na, eu, auth}, want: na},
		{name: "foreign host", ouCode: "EU", aud: []string{"https://fleet-api.prd.eu.example.com"}, want: na},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
				"ou_code": tc.ouCode,
				"aud":     tc.aud,
			}).SignedString([]byte("secret"))
			require.NoError(t, err)

			assert.Equal(t, tc.want, fleetBaseUrl(token))
		})
	}

	assert.Equal(t, na, fleetBaseUrl("access"), "not a jwt")
}
