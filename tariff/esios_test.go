package tariff

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/require"
)

// Canarias prices apply per local hour, one hour behind the peninsular
// instants REE publishes, for both indicators.
func TestEsiosCanariasShift(t *testing.T) {
	tmpl, err := templates.ByName(templates.Tariff, "esios")
	require.NoError(t, err)

	for _, tc := range []struct {
		region, indicator string
		shifted           bool
	}{
		{"Canarias", "1001", true},
		{"Canarias", "1739", true},
		{"Península", "1001", false},
	} {
		b, _, err := tmpl.RenderResult(templates.Tariff, templates.RenderModeInstance, map[string]any{
			"securitytoken": "token",
			"indicator":     tc.indicator,
			"region":        tc.region,
		})
		require.NoError(t, err)
		require.Equal(t, tc.shifted, strings.Contains(string(b), "mktime + 3600"), "%s %s", tc.region, tc.indicator)
	}
}
