package metrics

import (
	"math"
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

	// only slot 1 is priced: battery to home at feed-in 0.1, grid to loadpoint at 0.3
	assert.Equal(t, []Flow{
		{From: PV, To: Home, Energy: 0.5},
		{From: PV, To: Battery, Energy: 0.5},
		{From: PV, To: Export, Energy: 1},
		{From: Battery, To: Home, Energy: 0.5, Cost: 0.05, PricedEnergy: 0.5},
		{From: Grid, To: Loadpoint, Energy: 1, Cost: 0.3, PricedEnergy: 1},
	}, res.Flows)

	require.NotNil(t, res.Cost)
	assert.InDelta(t, 0.3, res.Cost.Import, 1e-9)
	assert.InDelta(t, 0.05, res.Cost.Export, 1e-9)
	assert.InDelta(t, 1, res.Cost.ImportEnergy, 1e-9)
	assert.InDelta(t, 0.5, res.Cost.ExportEnergy, 1e-9)
	assert.InDelta(t, 0.3, res.Cost.AvgGrid, 1e-9)

	assert.Nil(t, res.Co2)
}

func TestQueryFlowEntities(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	create := func(group, name string) entity {
		e := entity{Name: name, Group: group, Title: name}
		require.NoError(t, db.Instance.FirstOrCreate(&e, entity{Name: name}).Error)
		return e
	}
	pv, grid, home := create(PV, PV), create(Grid, Grid), create(Home, Home)
	carport, garage := create(Loadpoint, "carport"), create(Loadpoint, "garage")
	noon, evening := create(Consumer, "noon"), create(Consumer, "evening")
	noon2 := entity{Name: "noon2", Group: Consumer, Title: "noon"}
	require.NoError(t, db.Instance.Create(&noon2).Error)

	base := time.Date(2026, 9, 1, 12, 0, 0, 0, time.Now().Location())
	slot := func(i int) time.Time { return base.Add(time.Duration(i) * 15 * time.Minute) }
	tariff := func(i int, price float64) {
		feedin := 0.1
		require.NoError(t, PersistTariffs(slot(i), &price, &feedin, nil, nil))
	}

	// slot 0: solar covers home and carport, the noon consumer runs
	require.NoError(t, persist(pv, slot(0), 2, 0, nil, false))
	require.NoError(t, persist(home, slot(0), 1, 0, nil, false))
	require.NoError(t, persist(carport, slot(0), 1, 0, nil, false))
	require.NoError(t, persist(noon, slot(0), 0.4, 0, nil, false))
	tariff(0, 0.3)

	// slot 1: grid covers home and garage, the evening consumer runs
	require.NoError(t, persist(grid, slot(1), 2, 0, nil, false))
	require.NoError(t, persist(home, slot(1), 1, 0, nil, false))
	require.NoError(t, persist(garage, slot(1), 1, 0, nil, false))
	require.NoError(t, persist(evening, slot(1), 0.4, 0, nil, false))
	tariff(1, 0.4)

	// slot 2: both consumers exceed home, others clamps to zero
	require.NoError(t, persist(grid, slot(2), 0.5, 0, nil, false))
	require.NoError(t, persist(home, slot(2), 0.5, 0, nil, false))
	require.NoError(t, persist(noon, slot(2), 0.4, 0, nil, false))
	require.NoError(t, persist(evening, slot(2), 0.4, 0, nil, false))
	tariff(2, 0.2)

	// slot 3: grid covers half of home, a second "noon" meter runs
	require.NoError(t, persist(grid, slot(3), 0.5, 0, nil, false))
	require.NoError(t, persist(home, slot(3), 1, 0, nil, false))
	require.NoError(t, persist(noon2, slot(3), 0.4, 0, nil, false))
	tariff(3, 0.5)

	res, err := QueryFlow(slot(0), slot(4))
	require.NoError(t, err)

	round := func(flows []Flow) []Flow {
		for i := range flows {
			flows[i].Cost = math.Round(flows[i].Cost*1e9) / 1e9
			flows[i].PricedEnergy = math.Round(flows[i].PricedEnergy*1e9) / 1e9
		}
		return flows
	}

	// the sinks as before, consumers never enter the attribution
	assert.Equal(t, []Flow{
		{From: PV, To: Home, Energy: 1, Cost: 0.1, PricedEnergy: 1},
		{From: PV, To: Loadpoint, Energy: 1, Cost: 0.1, PricedEnergy: 1},
		{From: Grid, To: Home, Energy: 2, Cost: 0.75, PricedEnergy: 2},
		{From: Grid, To: Loadpoint, Energy: 1, Cost: 0.4, PricedEnergy: 1},
	}, round(res.Flows))

	// each entity priced by the slots it ran in, not the sink average, and only
	// for the part its sink's sources covered
	require.Len(t, res.Entities, 5)
	for i := range res.Entities {
		round(res.Entities[i].Flows)
	}
	assert.Equal(t, []EntityFlows{
		{Group: Consumer, Title: "evening", Flows: []Flow{
			{From: Grid, To: Home, Energy: 0.8, Cost: 0.24, PricedEnergy: 0.8},
		}},
		{Group: Consumer, Title: "noon", Flows: []Flow{
			{From: PV, To: Home, Energy: 0.4, Cost: 0.04, PricedEnergy: 0.4},
			{From: Grid, To: Home, Energy: 0.6, Cost: 0.18, PricedEnergy: 0.6},
		}},
		{Group: Loadpoint, Title: "carport", Flows: []Flow{
			{From: PV, To: Loadpoint, Energy: 1, Cost: 0.1, PricedEnergy: 1},
		}},
		{Group: Loadpoint, Title: "garage", Flows: []Flow{
			{From: Grid, To: Loadpoint, Energy: 1, Cost: 0.4, PricedEnergy: 1},
		}},
		{Group: Consumer, Virtual: true, Flows: []Flow{
			{From: PV, To: Home, Energy: 0.6, Cost: 0.06, PricedEnergy: 0.6},
			{From: Grid, To: Home, Energy: 0.9, Cost: 0.39, PricedEnergy: 0.9},
		}},
	}, res.Entities)
}
