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

	// fast charging accepts a higher peak
	lp.mode = api.ModeNow
	site.updateGridPeak()
	assert.Equal(t, 0.0, site.GetGridPeakTarget())

	// disabled
	lp.mode = api.ModePV
	require.NoError(t, site.SetGridPeakBaseline(0))
	site.updateGridPeak()
	assert.Equal(t, 0.0, site.GetGridPeakTarget())
}
