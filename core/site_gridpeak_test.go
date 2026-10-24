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
	assert.Equal(t, 0.0, p.peak(start))

	// 23:15-23:30: 10 min at 4 kW, 5 min at 1 kW
	assert.False(t, p.add(start.Add(18*time.Minute), 1000))
	assert.True(t, p.add(start.Add(23*time.Minute), 1000))
	assert.InDelta(t, 3000, p.peak(start), 1e-6)

	// export does not count, lower average does not raise
	assert.False(t, p.add(start.Add(30*time.Minute), -5000))
	assert.False(t, p.add(start.Add(38*time.Minute), 5000))
	assert.InDelta(t, 3000, p.peak(start), 1e-6)

	// 23:45-00:00 belongs to october
	assert.True(t, p.add(start.Add(53*time.Minute), 0))
	assert.InDelta(t, 5000, p.peak(start), 1e-6)

	// new month starts without peak
	nov := start.Add(53 * time.Minute)
	assert.Equal(t, 0.0, p.peak(nov))

	// gap invalidates the interval
	assert.False(t, p.add(nov.Add(time.Hour+5*time.Minute), 2000))
	assert.False(t, p.add(nov.Add(time.Hour+8*time.Minute), 2000))
	assert.Equal(t, 0.0, p.peak(nov))

	// 01:15-01:30 complete
	assert.False(t, p.add(nov.Add(time.Hour+23*time.Minute), 2000))
	assert.True(t, p.add(nov.Add(time.Hour+30*time.Minute), 2000))
	assert.InDelta(t, 2000, p.peak(nov), 1e-6)
	assert.Equal(t, nov.Add(time.Hour+15*time.Minute), p.Time)

	// gap ending on an interval boundary starts a complete interval
	assert.False(t, p.add(nov.Add(2*time.Hour), 3000))
	assert.True(t, p.add(nov.Add(2*time.Hour+15*time.Minute), 0))
	assert.InDelta(t, 3000, p.peak(nov), 1e-6)
}

func TestGridPeakTarget(t *testing.T) {
	t.Cleanup(func() {
		_ = settings.Delete(keys.Country)
		_ = settings.Delete(keys.GridPeakBaseline)
	})

	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)
	lp := &Loadpoint{mode: api.ModePV}
	site := &Site{
		log:        util.NewLogger("foo"),
		loadpoints: []*Loadpoint{lp},
		gridPeak:   gridPeak{Time: now.Add(-time.Hour), Power: 6000},
	}

	// no regime
	settings.SetString(keys.Country, "DE")
	require.NoError(t, site.SetGridPeakBaseline(4000))
	site.updateGridPeak(now)
	assert.Equal(t, 0.0, site.GetGridPeakTarget())

	// month's peak above baseline
	settings.SetString(keys.Country, "AT")
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
