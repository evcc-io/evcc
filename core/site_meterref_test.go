package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDropMissingMeterRefs verifies that references to meters which no longer
// exist are removed so the site can boot instead of crashing (see #31698).
func TestDropMissingMeterRefs(t *testing.T) {
	config.Reset()

	// register a single existing meter
	require.NoError(t, config.Meters().Add(
		config.NewStaticDevice(config.Named{Name: "db:1"}, api.Meter(nil)),
	))

	site := &Site{log: util.NewLogger("site")}
	site.Meters.GridMeterRef = "db:99"                  // missing
	site.Meters.PVMetersRef = []string{"db:1", "db:98"} // one existing, one missing
	site.Meters.BatteryMetersRef = []string{"db:1"}     // existing

	site.dropMissingMeterRefs()

	assert.Empty(t, site.Meters.GridMeterRef, "missing grid ref should be dropped")
	assert.Equal(t, []string{"db:1"}, site.Meters.PVMetersRef, "missing pv ref should be dropped")
	assert.Equal(t, []string{"db:1"}, site.Meters.BatteryMetersRef, "existing ref should be kept")
}
