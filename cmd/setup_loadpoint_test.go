package cmd

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestConfigureLoadpointsStalePhases asserts that a stored automatic phase mode
// no longer supported by the charger does not fail boot
func TestConfigureLoadpointsStalePhases(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	config.Reset()

	// charger without phase switching
	var charger api.Charger = api.NewMockCharger(gomock.NewController(t))
	require.NoError(t, config.Chargers().Add(config.NewStaticDevice(config.Named{Name: "test"}, charger)))

	conf, err := config.AddConfig(templates.Loadpoint, map[string]any{"charger": "test", "phasesConfigured": 0})
	require.NoError(t, err)
	t.Cleanup(func() { _ = config.Loadpoints().Delete(config.NameForID(conf.ID)) })

	require.NoError(t, configureLoadpoints(globalconfig.All{}))

	lp := config.Loadpoints().Devices()[0].Instance()
	require.Equal(t, 3, lp.GetPhasesConfigured())
}
