package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/require"
)

// TestFilterConfigurableCurtailers ensures curtailment device refs are not filtered
// against the meter registry, which would drop them from the persisted settings
func TestFilterConfigurableCurtailers(t *testing.T) {
	name := config.NameForID(1)

	conf := &config.Config{ID: 1}

	var m api.Curtailer = &curtailableMeter{}
	dev := config.NewConfigurableDevice(conf, m)
	require.NoError(t, config.Curtailers().Add(dev))
	t.Cleanup(func() { _ = config.Curtailers().Delete(name) })

	require.Equal(t, []string{name}, filterConfigurableCurtailers([]string{name}))
	require.Empty(t, filterConfigurableMeter([]string{name}), "curtailer must not resolve as meter")
}
