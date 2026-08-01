package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPowerwallTemplateOptionalFleetCredentials(t *testing.T) {
	tmpl, err := ByName(Meter, "tesla-powerwall")
	require.NoError(t, err)
	for _, name := range []string{"fleetClientId", "fleetAccessToken", "fleetRefreshToken"} {
		index, param := tmpl.ParamByName(name)
		require.NotEqual(t, -1, index, "%s must be defined", name)
		assert.False(t, param.IsRequired(), "%s must remain optional", name)
	}

	t.Run("local meter", func(t *testing.T) {
		rendered, _, err := tmpl.RenderResult(RenderModeInstance, map[string]any{
			"usage": "grid", "host": "powerwall.local", "password": "secret",
		})
		require.NoError(t, err)
		assert.NotContains(t, string(rendered), "credentials:")
		assert.NotContains(t, string(rendered), "tokens:")
	})

	t.Run("Fleet battery control", func(t *testing.T) {
		rendered, _, err := tmpl.RenderResult(RenderModeInstance, map[string]any{
			"usage": "battery", "host": "powerwall.local", "password": "secret",
			"fleetClientId": "client", "fleetAccessToken": "access", "fleetRefreshToken": "refresh",
		})
		require.NoError(t, err)
		assert.Contains(t, string(rendered), "credentials:\n  id: client")
		assert.Contains(t, string(rendered), "tokens:\n  access: access\n  refresh: refresh")
	})
}
