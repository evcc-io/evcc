package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGridPeakTarget(t *testing.T) {
	t.Cleanup(func() {
		_ = settings.Delete(keys.GridPeakBaseline)
	})

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, metrics.SetupSchema())

	// one completed slot at 6 kW
	clock := clock.NewMock()
	clock.Set(time.Date(2026, 10, 24, 12, 0, 0, 0, time.Local))
	col, err := metrics.NewCollector(metrics.Grid, "grid", "grid", metrics.WithClock(clock))
	require.NoError(t, err)
	require.NoError(t, col.AddEnergy(nil, nil, 6000))
	clock.Add(15 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 6000))

	lp := &Loadpoint{mode: api.ModePV}
	site := &Site{
		log:        util.NewLogger("foo"),
		loadpoints: []*Loadpoint{nil, lp}, // nil = disabled loadpoint
		Meters:     MetersConfig{GridMeterRef: "grid"},
		collectors: map[string]*metrics.Collector{"grid": col},
	}

	// month's peak above baseline
	require.NoError(t, site.SetGridPeakBaseline(4000))
	site.updateGridPeak()
	assert.InDelta(t, 6000, site.GetGridPeakTarget(), 1e-6)

	// baseline above month's peak
	require.NoError(t, site.SetGridPeakBaseline(8000))
	site.updateGridPeak()
	assert.Equal(t, 8000.0, site.GetGridPeakTarget())

	assert.Equal(t, 8000.0, site.getGridPeakLimit(), "unused slot")

	// 12 kW for 5 min uses half of the slot's 2 kWh, leaving 6 kW for the remaining 10 min
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 12000))
	site.updateGridPeak()
	assert.Equal(t, 8000.0, site.GetGridPeakTarget())
	assert.InDelta(t, 6000, site.getGridPeakLimit(), 1e-6)

	// time left is floored near the slot end
	clock.Add(9*time.Minute + 30*time.Second)
	require.NoError(t, col.AddEnergy(nil, nil, 1200))
	site.updateGridPeak()
	assert.InDelta(t, 8000, site.getGridPeakLimit(), 1e-6)

	// 2.19 kWh exceed the slot's budget: the running slot raises the target, nothing is left to import
	clock.Add(20 * time.Second)
	require.NoError(t, col.AddEnergy(nil, nil, 180000))
	site.updateGridPeak()
	assert.InDelta(t, 8760, site.GetGridPeakTarget(), 1e-6)
	assert.Equal(t, 1.0, site.getGridPeakLimit())

	// fast charging accepts a higher peak
	lp.mode = api.ModeNow
	site.updateGridPeak()
	assert.Equal(t, 0.0, site.GetGridPeakTarget())
	assert.Equal(t, 0.0, site.getGridPeakLimit())

	// disabled
	lp.mode = api.ModePV
	require.NoError(t, site.SetGridPeakBaseline(0))
	site.updateGridPeak()
	assert.Equal(t, 0.0, site.GetGridPeakTarget())
}
