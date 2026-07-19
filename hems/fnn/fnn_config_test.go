package fnn

import (
	"maps"
	"testing"

	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPowerParamFallback verifies productionnominalmax supersedes the
// deprecated maxcurtailpower/maxpower params while both keep working.
func TestPowerParamFallback(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))

	w3 := map[string]any{"source": "const", "value": "true"}

	for _, tc := range []struct {
		name string
		conf map[string]any
		want float64
	}{
		{"new param", map[string]any{"productionnominalmax": 5000.0}, 5000},
		{"deprecated maxcurtailpower", map[string]any{"maxcurtailpower": 4000.0}, 4000},
		{"deprecated maxpower", map[string]any{"maxpower": 3000.0}, 3000},
		{"new supersedes deprecated", map[string]any{
			"productionnominalmax": 5000.0, "maxcurtailpower": 4000.0, "maxpower": 3000.0,
		}, 5000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			conf := map[string]any{"w3": w3}
			maps.Copy(conf, tc.conf)

			fnn, err := NewFromConfig(t.Context(), conf, &stubSite{})
			require.NoError(t, err)
			assert.Equal(t, tc.want, fnn.productionNominalMax)
		})
	}
}
