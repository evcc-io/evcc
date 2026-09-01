package core

import (
	"testing"

	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSiteSettingsRoundtrip asserts that settings written through the API are
// restored into a fresh site sharing the same store.
func TestSiteSettingsRoundtrip(t *testing.T) {
	store := settings.NewMemorySettings()

	site := &Site{log: util.NewLogger("foo"), settings: store}
	require.NoError(t, site.SetResidualPower(150))
	require.NoError(t, site.SetGridExportLimit(7000))
	site.SetSolarAdjusted(true)
	require.NoError(t, site.SetOptimizerChargingStrategy(defaultOptimizerChargingStrategy))
	site.SetTitle("home")

	restored := &Site{log: util.NewLogger("bar"), settings: store}
	require.NoError(t, restored.restoreSettings())
	restored.restoreMetersAndTitle()

	assert.Equal(t, 150.0, restored.GetResidualPower())
	assert.Equal(t, 7000.0, restored.GetGridExportLimit())
	assert.True(t, restored.GetSolarAdjusted())
	assert.Equal(t, defaultOptimizerChargingStrategy, restored.GetOptimizerChargingStrategy())
	assert.Equal(t, "home", restored.GetTitle())
}
