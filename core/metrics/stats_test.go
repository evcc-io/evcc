package metrics

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectorEnergyStats(t *testing.T) {
	clock := clock.NewMock()

	loc := time.Now().Location()
	midnight := time.Date(2026, 4, 15, 0, 0, 0, 0, loc)
	clock.Set(midnight.Add(2*time.Hour + 5*time.Minute))

	slotStart := clock.Now().Truncate(15 * time.Minute)

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("stats", "stats", "", WithClock(clock))
	require.NoError(t, err)

	e := col.entity
	require.NoError(t, persist(e, slotStart.AddDate(0, 0, -8), 100, 0, nil, false))               // outside 7d
	require.NoError(t, persist(e, slotStart.Add(-24*time.Hour-15*time.Minute), 5, 0, nil, false)) // 7d only
	require.NoError(t, persist(e, slotStart.Add(-24*time.Hour), 7, 0, nil, false))                // 24h window start
	require.NoError(t, persist(e, slotStart.Add(-12*time.Hour), 3, 0, nil, false))                // yesterday, within 24h
	require.NoError(t, persist(e, midnight, 1, 0, nil, false))                                    // first slot today
	require.NoError(t, persist(e, slotStart.Add(-15*time.Minute), 2, 0, nil, false))              // last completed slot
	require.NoError(t, persist(e, slotStart, 4, 0, nil, false))                                   // current slot, excluded

	stats, err := col.EnergyStats()
	require.NoError(t, err)
	require.InDelta(t, 3, stats.Today, 1e-9)
	require.InDelta(t, 13, stats.Last24h, 1e-9)
	require.InDelta(t, 18, stats.Last7d, 1e-9)

	// in-flight accumulator counts towards today only
	col.accu.Energy = 0.5
	stats, err = col.EnergyStats()
	require.NoError(t, err)
	require.InDelta(t, 3.5, stats.Today, 1e-9)
	require.InDelta(t, 13, stats.Last24h, 1e-9)

	// next slot: cache refreshes, windows shift by one slot
	clock.Add(15 * time.Minute)
	stats, err = col.EnergyStats()
	require.NoError(t, err)
	require.InDelta(t, 7.5, stats.Today, 1e-9)  // 1+2+4 + in-flight
	require.InDelta(t, 10, stats.Last24h, 1e-9) // drops 7, gains 4
	require.InDelta(t, 22, stats.Last7d, 1e-9)  // gains 4, 8d slot stays out
}

func TestCollectorDailyPeakEnergy(t *testing.T) {
	clock := clock.NewMock()

	loc := time.Now().Location()
	day := func(d int, hour, min int) time.Time {
		return time.Date(2026, 11, d, hour, min, 0, 0, loc)
	}
	clock.Set(day(3, 18, 7))

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector(Grid, "grid", "", WithClock(clock))
	require.NoError(t, err)

	// replaced grid meter earlier in the month, same group
	old := entity{Group: Grid, Name: "grid-old"}
	require.NoError(t, db.Instance.Create(&old).Error)
	require.NoError(t, persist(old, day(1, 9, 0), 3.0, 0, nil, false))

	e := col.entity
	require.NoError(t, persist(e, day(1, 10, 0), 1.0, 0, nil, false))
	require.NoError(t, persist(e, day(1, 10, 15), 2.0, 0, nil, false))
	require.NoError(t, persist(e, day(3, 0, 0), 0.5, 0, nil, false))
	require.NoError(t, persist(e, day(3, 1, 0), 9.0, 0, nil, true))                              // recovered, excluded
	require.NoError(t, persist(e, time.Date(2026, 10, 31, 23, 45, 0, 0, loc), 7, 0, nil, false)) // previous month
	require.NoError(t, persist(e, day(3, 18, 0), 9.0, 0, nil, false))                            // current slot, excluded

	days, err := col.DailyPeakEnergy()
	require.NoError(t, err)
	require.Len(t, days, 3)
	assert.InDelta(t, 3.0, *days[0], 1e-9)
	assert.Nil(t, days[1])
	assert.InDelta(t, 0.5, *days[2], 1e-9)

	// in-flight slot raises today once it exceeds the day's peak
	require.NoError(t, col.AddEnergy(nil, nil, 9000))
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 9000))

	days, err = col.DailyPeakEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 0.75, *days[2], 1e-9)

	// restored collector with skipped slots: in-flight energy is downtime catchup, not a peak
	e2 := entity{Group: Grid, Name: "grid2"}
	require.NoError(t, db.Instance.Create(&e2).Error)
	require.NoError(t, db.Instance.Model(&e2).UpdateColumns(map[string]any{"energy_meter": 100.0, "return_energy_meter": 10.0}).Error)
	require.NoError(t, persist(e2, day(3, 0, 0), 0.5, 0, nil, false))

	col2, err := NewCollector(Grid, "grid2", "", WithClock(clock))
	require.NoError(t, err)
	require.NoError(t, col2.SetEnergyMeterTotal(105))

	days, err = col2.DailyPeakEnergy()
	require.NoError(t, err)
	assert.InDelta(t, 0.5, *days[2], 1e-9)
}
