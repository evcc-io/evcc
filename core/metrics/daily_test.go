package metrics

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDailyRollup(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	entities := make(map[string]entity)
	for _, group := range []string{PV, Grid, Loadpoint} {
		e := entity{Name: group, Group: group}
		require.NoError(t, db.Instance.Create(&e).Error)
		entities[group] = e
	}
	entities[Home] = entity{Id: 1, Group: Home}

	// three days of slots, the first one starts mid day
	var slots float64
	from := time.Date(2025, 3, 29, 0, 0, 0, 0, time.Local)
	to := from.AddDate(0, 0, 3)
	for ts := from.Add(6 * time.Hour); ts.Before(to); ts = ts.Add(15 * time.Minute) {
		slots++
		require.NoError(t, persist(entities[PV], ts, 1, 0, nil, false))
		require.NoError(t, persist(entities[Home], ts, 0.5, 0, nil, false))
		require.NoError(t, persist(entities[Loadpoint], ts, 1, 0, nil, false))
		require.NoError(t, persist(entities[Grid], ts, 0.75, 0.25, nil, false))
		// binary fractions add up exactly in any order
		require.NoError(t, PersistTariffs(ts, new(0.25), new(0.125), new(512.0), nil))
	}

	query := func() ([]Series, FlowResult) {
		energy, err := QueryEnergy(from.Add(time.Hour), to, "month", false)
		require.NoError(t, err)
		flow, err := QueryFlow(from.Add(time.Hour), to)
		require.NoError(t, err)
		flow.entities = nil
		return energy, flow
	}

	// slots only
	dailyNow = func() time.Time { return from }
	t.Cleanup(func() { dailyNow = time.Now })

	energy, flow := query()
	require.Zero(t, dailyEnd.Load())
	require.NotEmpty(t, flow.Entities)

	// rolled up days in between the partial first day and the slots after
	dailyNow = func() time.Time { return to.Add(-time.Hour) }
	rolledEnergy, rolledFlow := query()
	require.Equal(t, to.AddDate(0, 0, -1).Unix(), dailyEnd.Load())

	var days int64
	require.NoError(t, db.Instance.Model(new(flowDay)).Count(&days).Error)
	require.Equal(t, int64(2), days)

	assert.Equal(t, energy, rolledEnergy)
	assert.Equal(t, flow, rolledFlow)

	pv := func() float64 {
		energy, _ := query()
		for _, s := range energy {
			if s.Group == PV {
				return s.Data[0].Energy
			}
		}
		return 0
	}

	// a slot written into a rolled up day is not hidden by the rollup
	dailyNow = func() time.Time { return to.AddDate(0, 0, 1) }
	require.NoError(t, persist(entities[PV], from.Add(3*time.Hour), 4, 0, nil, false))
	assert.Equal(t, slots+4, pv())
	require.Equal(t, to.Unix(), dailyEnd.Load())

	// neither are deleted slots
	second := from.AddDate(0, 0, 1)
	_, err := DeleteEnergy(second, second.Add(time.Hour), EnergyFilter{Group: PV})
	require.NoError(t, err)
	require.Equal(t, second.Unix(), dailyEnd.Load())
	assert.Equal(t, slots, pv())

	// days another process rolled up ahead of this one are replaced
	dailyEnd.Store(second.Unix())
	third := second.AddDate(0, 0, 1)
	_, err = DeleteEnergy(third, third.Add(time.Hour), EnergyFilter{Group: PV})
	require.NoError(t, err)
	slots -= 4
	assert.Equal(t, slots, pv())
	require.Equal(t, to.Unix(), dailyEnd.Load())
}

func TestBuckets(t *testing.T) {
	local := time.Local
	t.Cleanup(func() { time.Local = local })

	hours := func(buckets [][2]int64) []float64 {
		var res []float64
		for _, b := range buckets {
			res = append(res, time.Unix(b[1], 0).Sub(time.Unix(b[0], 0)).Hours())
		}
		return res
	}

	for _, tc := range []struct {
		name, zone, from, to, aggregate string
		want                            []float64
	}{
		{"daylight saving starts", "Europe/Berlin", "2025-03-29 06:15", "2025-04-01 00:00", "day", []float64{17.75, 23, 24}},
		{"daylight saving ends", "Europe/Berlin", "2025-10-26 00:00", "2025-10-27 12:00", "day", []float64{25, 12}},
		{"repeated hour is one bucket", "Europe/Berlin", "2025-10-26 01:00", "2025-10-26 04:00", "hour", []float64{1, 2, 1}},
		{"months", "Europe/Berlin", "2025-01-15 00:00", "2025-04-01 00:00", "month", []float64{17 * 24, 28 * 24, 31*24 - 1}},
		{"midnight skipped, west of UTC", "America/Santiago", "2024-09-07 00:00", "2024-09-10 00:00", "day", []float64{24, 23, 24}},
		{"midnight skipped, east of UTC", "Asia/Beirut", "2025-03-29 00:00", "2025-04-01 00:00", "day", []float64{24, 23, 24}},
		{"half hour shift", "Australia/Lord_Howe", "2025-04-06 00:00", "2025-04-06 04:00", "hour", []float64{1, 1.5, 1, 1}},
		{"quarter hour offset", "Asia/Kathmandu", "2025-04-06 00:30", "2025-04-06 03:00", "hour", []float64{0.5, 1, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			time.Local, err = time.LoadLocation(tc.zone)
			require.NoError(t, err)

			from, err := time.ParseInLocation("2006-01-02 15:04", tc.from, time.Local)
			require.NoError(t, err)
			to, err := time.ParseInLocation("2006-01-02 15:04", tc.to, time.Local)
			require.NoError(t, err)

			res := buckets(from, to, tc.aggregate)
			assert.Equal(t, tc.want, hours(res))
			assert.Equal(t, from.Unix(), res[0][0])
			assert.Equal(t, to.Unix(), res[len(res)-1][1])

			// a slot inside a bucket aligns to the start of that bucket
			for _, b := range res[1:] {
				assert.Equal(t, b[0], aggregateStarts[tc.aggregate](time.Unix(b[1]-900, 0)).Unix())
			}
		})
	}

	// only whole days below the rollup end are read from it
	time.Local = local
	from := time.Date(2025, 3, 29, 6, 15, 0, 0, time.Local)
	day := dayStart(from)
	d0, d1 := wholeDays(from, day.AddDate(0, 0, 3), day.AddDate(0, 0, 2).Unix())
	assert.Equal(t, [2]int64{day.AddDate(0, 0, 1).Unix(), day.AddDate(0, 0, 2).Unix()}, [2]int64{d0, d1})
}
