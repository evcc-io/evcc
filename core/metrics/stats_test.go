package metrics

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/db"
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

	col, err := NewCollector(Loadpoint, "stats", "", WithClock(clock))
	require.NoError(t, err)

	e := col.entity
	require.NoError(t, persist(e, slotStart.AddDate(0, 0, -8), 100, 0, nil, false))               // outside 7d
	require.NoError(t, persist(e, slotStart.Add(-24*time.Hour-15*time.Minute), 5, 0, nil, false)) // 7d only
	require.NoError(t, persist(e, slotStart.Add(-24*time.Hour), 7, 0, nil, false))                // 24h window start
	require.NoError(t, persist(e, slotStart.Add(-12*time.Hour), 3, 0, nil, false))                // yesterday, within 24h
	require.NoError(t, persist(e, midnight, 1, 0, nil, false))                                    // first slot today
	require.NoError(t, persist(e, slotStart.Add(-15*time.Minute), 2, 0, nil, false))              // last completed slot
	require.NoError(t, persist(e, slotStart, 4, 0, nil, false))                                   // current slot, excluded

	// sources and tariffs for three slots, the rest stays unattributed
	pv, err := createEntity(PV, PV, "")
	require.NoError(t, err)
	grid, err := createEntity(Grid, Grid, "")
	require.NoError(t, err)
	feedin := 0.1
	tariff := func(ts time.Time, price float64) {
		require.NoError(t, PersistTariffs(ts, &price, &feedin, nil, nil))
	}
	require.NoError(t, persist(grid, slotStart.Add(-12*time.Hour), 3, 0, nil, false)) // grid, no tariff
	require.NoError(t, persist(pv, midnight, 1, 0, nil, false))                       // solar at feed-in price
	tariff(midnight, 0.3)
	require.NoError(t, persist(grid, slotStart.Add(-15*time.Minute), 2, 0, nil, false)) // grid at grid price
	tariff(slotStart.Add(-15*time.Minute), 0.4)

	stats, err := col.EnergyStats()
	require.NoError(t, err)
	require.InDelta(t, 3000, stats.Today, 1e-9)
	require.InDelta(t, 13000, stats.Last24h.Energy, 1e-9)
	require.InDelta(t, 18000, stats.Last7d.Energy, 1e-9)

	// 1 kWh pv at 0.1, 2 kWh grid at 0.4, 3 kWh unpriced grid,
	// the 7 kWh slot without sources stays unattributed
	require.InDelta(t, 100.0/6, *stats.Last24h.Autarky, 1e-9)
	require.InDelta(t, 0.9, *stats.Last24h.Price, 1e-9)
	require.InDelta(t, 0.3, *stats.Last24h.PricePerKWh, 1e-9)
	require.InDelta(t, 100.0/6, *stats.Last7d.Autarky, 1e-9)

	// in-flight accumulator counts towards today only
	col.accu.Energy = 0.5
	stats, err = col.EnergyStats()
	require.NoError(t, err)
	require.InDelta(t, 3500, stats.Today, 1e-9)
	require.InDelta(t, 13000, stats.Last24h.Energy, 1e-9)

	// next slot: cache refreshes, windows shift by one slot
	clock.Add(15 * time.Minute)
	stats, err = col.EnergyStats()
	require.NoError(t, err)
	require.InDelta(t, 7500, stats.Today, 1e-9)           // 1+2+4 + in-flight
	require.InDelta(t, 10000, stats.Last24h.Energy, 1e-9) // drops 7, gains 4
	require.InDelta(t, 22000, stats.Last7d.Energy, 1e-9)  // gains 4, 8d slot stays out
}

func TestPeriodStatsEmpty(t *testing.T) {
	require.Equal(t, PeriodStats{Energy: 1000}, (&flowEntity{metered: 1}).stats())
}
