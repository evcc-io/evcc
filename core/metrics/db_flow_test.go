package metrics

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func collectFlows(s flowSlot) map[string]float64 {
	res := make(map[string]float64)
	attribute(s, func(from, to string, energy float64) {
		res[from+">"+to] += energy
	})
	return res
}

func TestAttribute(t *testing.T) {
	for _, tc := range []struct {
		name string
		slot flowSlot
		want map[string]float64
	}{
		{
			"pv covers home, loadpoint and export",
			flowSlot{PV: 10, Home: 2, Loadpoint: 3, GridOut: 5},
			map[string]float64{"pv>home": 2, "pv>loadpoint": 3, "pv>export": 5},
		},
		{
			"battery charge before loadpoints, grid fills the rest",
			flowSlot{PV: 4, Home: 2, BatIn: 3, Loadpoint: 6, GridIn: 7},
			map[string]float64{"pv>home": 2, "pv>battery": 2, "grid>battery": 1, "grid>loadpoint": 6},
		},
		{
			"battery discharge after pv, never to battery",
			flowSlot{PV: 1, BatOut: 3, BatIn: 1, Home: 2, Loadpoint: 1, GridIn: 1},
			map[string]float64{"pv>home": 1, "battery>home": 1, "battery>loadpoint": 1, "grid>battery": 1},
		},
		{
			"grid never exports, mismatch dropped",
			flowSlot{GridIn: 2, GridOut: 1, Home: 1},
			map[string]float64{"grid>home": 1},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, collectFlows(tc.slot))
		})
	}
}

func TestQueryFlow(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	entities := make(map[string]entity)
	for _, group := range []string{PV, Grid, Battery, Home, Loadpoint} {
		e := entity{Name: group, Group: group}
		require.NoError(t, db.Instance.FirstOrCreate(&e, entity{Name: group}).Error)
		entities[group] = e
	}

	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.Now().Location())
	slot := func(i int) time.Time { return base.Add(time.Duration(i) * 15 * time.Minute) }

	// slot 0: sunny, export; slot 1: night, grid import with unattributed export; slot 2: outside range
	require.NoError(t, persist(entities[PV], slot(0), 2, 0, nil, false))
	require.NoError(t, persist(entities[Home], slot(0), 0.5, 0, nil, false))
	require.NoError(t, persist(entities[Battery], slot(0), 0.5, 0, nil, false))
	require.NoError(t, persist(entities[Grid], slot(0), 0, 1, nil, false))

	require.NoError(t, persist(entities[Grid], slot(1), 1, 0.5, nil, false))
	require.NoError(t, persist(entities[Battery], slot(1), 0, 0.5, nil, false))
	require.NoError(t, persist(entities[Home], slot(1), 0.5, 0, nil, false))
	require.NoError(t, persist(entities[Loadpoint], slot(1), 1, 0, nil, false))

	require.NoError(t, persist(entities[Grid], slot(2), 5, 0, nil, false))
	require.NoError(t, persist(entities[Home], slot(2), 5, 0, nil, false))

	// tariffs only for slot 1
	grid, feedin := 0.3, 0.1
	require.NoError(t, PersistTariffs(slot(1), &grid, &feedin, nil, nil))

	res, err := QueryFlow(slot(0), slot(2))
	require.NoError(t, err)

	assert.Equal(t, []Flow{
		{PV, Home, 0.5},
		{PV, Battery, 0.5},
		{PV, Export, 1},
		{Battery, Home, 0.5},
		{Grid, Loadpoint, 1},
	}, res.Flows)

	require.NotNil(t, res.Cost)
	assert.InDelta(t, 0.3, res.Cost.Import, 1e-9)
	assert.InDelta(t, 0.05, res.Cost.Export, 1e-9)
	assert.InDelta(t, 1, res.Cost.ImportEnergy, 1e-9)
	assert.InDelta(t, 0.5, res.Cost.ExportEnergy, 1e-9)
	assert.InDelta(t, 0.3, res.Cost.AvgGrid, 1e-9)
	assert.InDelta(t, 0.3*1.5, res.Cost.Baseline, 1e-9) // slot 1 consumption only
	assert.InDelta(t, 1.5, res.Cost.ConsumptionEnergy, 1e-9)
	// grid 1 kWh to loadpoint at 0.3, battery 0.5 kWh to home at feed-in 0.1
	assert.InDelta(t, 0.3+0.05, res.Cost.Consumption, 1e-9)
	assert.InDelta(t, 0.05, res.Cost.Home, 1e-9)
	assert.InDelta(t, 0.5, res.Cost.HomeEnergy, 1e-9)

	assert.Nil(t, res.Co2)
}
