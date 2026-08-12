package metrics

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/server/db"
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
