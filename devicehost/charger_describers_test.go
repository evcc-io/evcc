package devicehost_test

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/devicehost"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestChargerDescribers(t *testing.T) {
	for _, tc := range []struct {
		name         string
		capabilities []string
		features     bool
		icon         bool
	}{
		{name: "none"},
		{name: "features", capabilities: []string{"api.FeatureDescriber"}, features: true},
		{name: "icon", capabilities: []string{"api.IconDescriber"}, icon: true},
		{name: "both", capabilities: []string{"api.FeatureDescriber", "api.IconDescriber"}, features: true, icon: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a host advertising this charger's optional describers.
			ctx := context.Background()
			host, err := devicehost.New(ctx, "describers", serve(t, tc.capabilities...))
			require.NoError(t, err)
			t.Cleanup(func() { require.NoError(t, host.Close()) })

			// When the charger is instantiated through its registered template.
			c, err := charger.NewFromConfig(ctx, "template", map[string]any{
				"template": "describers-wallbox",
				"uri":      "http://demo/wb",
			})
			require.NoError(t, err)

			// Then only advertised describers expose the host's metadata.
			fd, ok := api.Cap[api.FeatureDescriber](c)
			require.Equal(t, tc.features, ok)
			if ok {
				assert.Equal(t, []api.Feature{api.IntegratedDevice, api.Heating}, fd.Features())
			}

			id, ok := api.Cap[api.IconDescriber](c)
			require.Equal(t, tc.icon, ok)
			if ok {
				assert.Equal(t, "heatpump", id.Icon())
			}
		})
	}
}
