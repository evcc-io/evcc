package metrics

import (
	"math"
	"testing"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

// rolled returns the days that are rolled up
func rolled(t *testing.T) []int64 {
	var days []int64
	require.NoError(t, db.Instance.Model(new(flowDay)).Order("day").Pluck("day", &days).Error)
	return days
}

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
		// binary fractions add up exactly in any order, also the co2 per kWh over 1e3
		require.NoError(t, PersistTariffs(ts, new(0.25), new(0.125), new(500.0), nil))
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
	require.NoError(t, rollupDays(from))
	energy, flow := query()
	require.Empty(t, rolled(t), "a read does not roll up")
	require.NotEmpty(t, flow.Entities)

	// rolled up days in between the partial first day and the slots after
	second := nextDay(from)
	third := nextDay(second)
	require.NoError(t, rollupDays(to.Add(-time.Hour)))
	rolledEnergy, rolledFlow := query()
	require.Equal(t, []int64{from.Unix(), second.Unix()}, rolled(t))

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

	// a slot written into a rolled up day drops only that day
	now := to.AddDate(0, 0, 1)
	require.NoError(t, persist(entities[PV], from.Add(3*time.Hour), 4, 0, nil, false))
	require.Equal(t, []int64{second.Unix()}, rolled(t))
	assert.Equal(t, slots+4, pv())
	require.NoError(t, rollupDays(now))
	assert.Equal(t, slots+4, pv())
	require.Equal(t, []int64{from.Unix(), second.Unix(), third.Unix()}, rolled(t))

	// so do deleted slots, the day in between is read from the slots
	_, err := DeleteEnergy(second, second.Add(time.Hour), EnergyFilter{Group: PV})
	require.NoError(t, err)
	require.Equal(t, []int64{from.Unix(), third.Unix()}, rolled(t))
	assert.Equal(t, slots, pv())
	_, holeFlow := query()

	// sums left behind without their flow row are replaced
	require.NoError(t, db.Instance.Create(&energySum{Start: second.Unix(), Meter: entities[PV].Id, Energy: 1000}).Error)
	require.NoError(t, rollupDays(now))
	assert.Equal(t, slots, pv())
	_, filledFlow := query()
	assert.Equal(t, holeFlow, filledFlow)
	require.Len(t, rolled(t), 3)

	// rows off the day grid, as left by another time zone, are not read
	off := second.Add(time.Hour).Unix()
	require.NoError(t, db.Instance.Create(&flowDay{Day: off, Data: "{}"}).Error)
	require.NoError(t, db.Instance.Create(&energySum{Start: off, Meter: entities[PV].Id, Energy: 1000}).Error)
	assert.Equal(t, slots, pv())
}

func TestWriteDays(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	pv := entity{Name: PV, Group: PV}
	require.NoError(t, db.Instance.Create(&pv).Error)

	now := time.Now()
	today := dayStart(now).Unix()
	old := dayStart(now.AddDate(0, 0, -7))
	for _, day := range []int64{old.Unix(), today} {
		require.NoError(t, db.Instance.Create(&flowDay{Day: day, Data: "{}"}).Error)
	}

	// the current slot leaves the rollup alone
	require.NoError(t, persist(pv, now, 1, 0, nil, false))
	require.Len(t, rolled(t), 2)

	// a completed day loses its rollup, a failed change keeps it
	require.Error(t, writeDays(old, nextDay(old), func(*gorm.DB) error { return assert.AnError }))
	require.Len(t, rolled(t), 2)
	require.NoError(t, persist(pv, old, 1, 0, nil, false))
	require.Equal(t, []int64{today}, rolled(t))
}

func TestDailyRollupMidnightSkipped(t *testing.T) {
	local := time.Local
	t.Cleanup(func() { time.Local = local })

	var err error
	time.Local, err = time.LoadLocation("America/Santiago")
	require.NoError(t, err)

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	pv := entity{Name: PV, Group: PV}
	require.NoError(t, db.Instance.Create(&pv).Error)

	// the first chunk ends on 2024-09-08, which starts at 01:00
	from := time.Date(2024, 9, 8-dailyChunk, 0, 0, 0, 0, time.Local)
	skipped := localDate(2024, 9, 8)
	to := time.Date(2024, 9, 10, 0, 0, 0, 0, time.Local)
	require.Equal(t, 1, skipped.Hour())

	for ts := from; ts.Before(to); ts = ts.Add(15 * time.Minute) {
		require.NoError(t, persist(pv, ts, 1, 0, nil, false))
	}

	require.NoError(t, rollupDays(to.Add(12*time.Hour)))

	days := rolled(t)
	require.Len(t, days, dailyChunk+2)
	for _, d := range days {
		day := time.Unix(d, 0)
		assert.True(t, day.Equal(dayStart(day)), day)
	}

	// a restart keeps the rollup, the missing days start at the skipped midnight
	require.NoError(t, deleteDaily(db.Instance, skipped.Unix(), math.MaxInt64))
	require.NoError(t, loadDaily())
	require.Len(t, rolled(t), dailyChunk)

	missing, err := missingDays(to)
	require.NoError(t, err)
	require.Len(t, missing, 2)
	assert.Equal(t, skipped.Unix(), missing[0][0])
	assert.True(t, wholeDay(missing[0]))
	assert.False(t, wholeDay([2]int64{skipped.Add(-time.Hour).Unix(), missing[0][1]}))
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

	// only days that are inside the period from start to end are read from the rollup
	time.Local = local
	from := time.Date(2025, 3, 29, 6, 15, 0, 0, time.Local)
	var whole []bool
	for _, b := range buckets(from, nextDay(nextDay(dayStart(from))).Add(time.Hour), "day") {
		whole = append(whole, wholeDay(b))
	}
	assert.Equal(t, []bool{false, true, false}, whole)
}
