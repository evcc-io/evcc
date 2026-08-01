package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPowerwallTemplates(t *testing.T) {
	local, err := ByName(Meter, "tesla-powerwall")
	require.NoError(t, err)
	localConfig, _, err := local.RenderResult(RenderModeInstance, map[string]any{
		"usage": "grid", "host": "powerwall.local", "password": "secret",
	})
	require.NoError(t, err)
	assert.Contains(t, string(localConfig), "type: powerwall")
	assert.NotContains(t, string(localConfig), "credentials:")
	for _, name := range []string{"fleetClientId", "fleetAccessToken", "fleetRefreshToken"} {
		_, param := local.ParamByName(name)
		assert.Empty(t, param.Name)
	}

	fleet, err := ByName(Meter, "tesla-powerwall-fleet")
	require.NoError(t, err)
	fleetConfig, _, err := fleet.RenderResult(RenderModeInstance, map[string]any{
		"host": "powerwall.local", "password": "secret", "fleetClientId": "client",
		"fleetAccessToken": "access", "fleetRefreshToken": "refresh",
	})
	require.NoError(t, err)
	assert.Contains(t, string(fleetConfig), "type: powerwall-fleet")
	assert.Contains(t, string(fleetConfig), "usage: battery")
	for _, name := range []string{"fleetClientId", "fleetAccessToken", "fleetRefreshToken"} {
		_, param := fleet.ParamByName(name)
		assert.True(t, param.Required, "%s must be required", name)
	}
}
