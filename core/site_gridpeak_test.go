package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGridPeak(t *testing.T) {
	var p gridPeak
	start := time.Date(2026, 10, 31, 23, 7, 0, 0, time.UTC)

	// partial interval after start is not recorded
	assert.False(t, p.add(start, 8000))
	assert.False(t, p.add(start.Add(8*time.Minute), 4000))
	assert.Equal(t, 0.0, p.state(start).peak())

	// 23:15-23:30: 10 min at 4 kW, 5 min at 1 kW
	assert.False(t, p.add(start.Add(18*time.Minute), 1000))
	assert.True(t, p.add(start.Add(23*time.Minute), 1000))
	assert.InDelta(t, 3000, p.state(start).peak(), 1e-6)

	// export does not count, lower average does not raise
	assert.False(t, p.add(start.Add(30*time.Minute), -5000))
	assert.False(t, p.add(start.Add(38*time.Minute), 5000))
	assert.InDelta(t, 3000, p.state(start).peak(), 1e-6)

	// 23:45-00:00 belongs to october
	assert.True(t, p.add(start.Add(53*time.Minute), 0))
	assert.InDelta(t, 5000, p.state(start).peak(), 1e-6)

	// new month starts without peak
	nov := start.Add(53 * time.Minute)
	assert.Equal(t, 0.0, p.state(nov).peak())
	assert.Equal(t, gridPeak{Days: []float64{}}, p.state(nov))

	// gap invalidates the interval
	assert.False(t, p.add(nov.Add(time.Hour+5*time.Minute), 2000))
	assert.False(t, p.add(nov.Add(time.Hour+8*time.Minute), 2000))
	assert.Equal(t, 0.0, p.state(nov).peak())

	// 01:15-01:30 complete
	assert.False(t, p.add(nov.Add(time.Hour+23*time.Minute), 2000))
	assert.True(t, p.add(nov.Add(time.Hour+30*time.Minute), 2000))
	assert.InDelta(t, 2000, p.state(nov).peak(), 1e-6)
	assert.Equal(t, nov.Add(time.Hour+15*time.Minute), p.Time)

	// gap ending on an interval boundary starts a complete interval
	assert.False(t, p.add(nov.Add(2*time.Hour), 3000))
	assert.True(t, p.add(nov.Add(2*time.Hour+15*time.Minute), 0))
	assert.InDelta(t, 3000, p.state(nov).peak(), 1e-6)
}

func TestGridPeakDays(t *testing.T) {
	var p gridPeak
	day := time.Date(2026, 11, 3, 18, 0, 0, 0, time.UTC)

	assert.False(t, p.add(day, 6000))
	assert.True(t, p.add(day.Add(15*time.Minute), 9000))
	assert.True(t, p.add(day.Add(30*time.Minute), 0))
	assert.False(t, p.add(day.Add(45*time.Minute), 0)) // lower average keeps the day's peak
	assert.Equal(t, []float64{0, 0, 9000}, p.Days)

	// next day records its own peak below the month's peak
	next := day.Add(24 * time.Hour)
	assert.False(t, p.add(next, 5000))
	assert.True(t, p.add(next.Add(15*time.Minute), 0))
	assert.Equal(t, []float64{0, 0, 9000, 5000}, p.Days)
	assert.InDelta(t, 9000, p.peak(), 1e-6)
}

func TestGridPeakTarget(t *testing.T) {
	t.Cleanup(func() {
		_ = settings.Delete(keys.GridPeakBaseline)
	})

	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	lp := &Loadpoint{mode: api.ModePV}
	site := &Site{
		log:        util.NewLogger("foo"),
		loadpoints: []*Loadpoint{nil, lp}, // nil = disabled loadpoint
		gridPeak:   gridPeak{Time: now.Add(-time.Hour), Days: []float64{23: 6000}},
	}

	// month's peak above baseline
	require.NoError(t, site.SetGridPeakBaseline(4000))
	site.updateGridPeak(now)
	assert.Equal(t, 6000.0, site.GetGridPeakTarget())

	// baseline above month's peak
	require.NoError(t, site.SetGridPeakBaseline(8000))
	site.updateGridPeak(now)
	assert.Equal(t, 8000.0, site.GetGridPeakTarget())

	// fast charging accepts a higher peak
	lp.mode = api.ModeNow
	site.updateGridPeak(now)
	assert.Equal(t, 0.0, site.GetGridPeakTarget())

	// disabled
	lp.mode = api.ModePV
	require.NoError(t, site.SetGridPeakBaseline(0))
	site.updateGridPeak(now)
	assert.Equal(t, 0.0, site.GetGridPeakTarget())
}
