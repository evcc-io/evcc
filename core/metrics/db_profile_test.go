package metrics

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/require"
)

func TestEnergyProfileWeekday(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	e := entity{Id: 2, Name: "pv1", Group: PV}
	require.NoError(t, db.Instance.Create(&e).Error)

	// 4 weeks of full days, today's weekday carries a distinct energy value
	today := time.Now().Weekday()
	for day := -28; day < 0; day++ {
		base := now.BeginningOfDay().AddDate(0, 0, day)

		energy := 1.0
		if base.Weekday() == today {
			energy = 2.0
		}

		for slot := range 96 {
			ts := base.Add(time.Duration(slot) * tariff.SlotDuration)
			require.NoError(t, persist(e, ts, energy, 0, nil, false))
		}
	}

	weekday := int(today)
	res, err := energyProfileFiltered(e, now.BeginningOfDay().AddDate(0, 0, -28), &weekday, 0.5)
	require.NoError(t, err)

	// only same-weekday slots must be averaged
	for i, v := range res {
		require.Equal(t, 2.0, v, "slot %d", i)
	}
}

func TestEnergyProfileTemperatureBinned(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	e := entity{Id: 3, Name: "heater1", Group: Loadpoint}
	require.NoError(t, db.Instance.Create(&e).Error)

	// Populate 14 days (2 samples per temperature bin)
	for day := -14; day < 0; day++ {
		base := now.BeginningOfDay().AddDate(0, 0, day)
		temp := float64(day%7 + 5) // temperatures repeat across the 2 weeks

		for slot := range 96 {
			ts := base.Add(time.Duration(slot) * tariff.SlotDuration)
			energy := 0.5 - float64(day%7)*0.05

			require.NoError(t, persist(e, ts, energy, 0, nil, false))

			tv := tariffValue{
				Timestamp:   ts.Unix(),
				Temperature: &temp,
			}
			require.NoError(t, db.Instance.Create(&tv).Error)
		}
	}

	// Populate a single-sample bin (temp=20°C on only 1 day); should be excluded by count >= 2
	singleBase := now.BeginningOfDay().AddDate(0, 0, -20)
	singleTemp := 20.0
	for slot := range 96 {
		ts := singleBase.Add(time.Duration(slot) * tariff.SlotDuration)
		require.NoError(t, persist(e, ts, 1.0, 0, nil, false))
		tv := tariffValue{
			Timestamp:   ts.Unix(),
			Temperature: &singleTemp,
		}
		require.NoError(t, db.Instance.Create(&tv).Error)
	}

	// Populate data older than 1 year (day=-400) at 10°C; should be ignored
	oldBase := now.BeginningOfDay().AddDate(0, 0, -400)
	oldTemp := 10.0
	for slot := range 96 {
		ts := oldBase.Add(time.Duration(slot) * tariff.SlotDuration)
		require.NoError(t, persist(e, ts, 99.0, 0, nil, false))
		tv := tariffValue{
			Timestamp:   ts.Unix(),
			Temperature: &oldTemp,
		}
		require.NoError(t, db.Instance.Create(&tv).Error)
	}

	res, err := energyProfileTemperatureBinned(e)
	require.NoError(t, err)
	require.Len(t, res, 96)

	// Verify that temperature bins exist for slot 0
	require.NotEmpty(t, res[0])
	require.Contains(t, res[0], 4)
	require.InDelta(t, 0.55, res[0][4], 1e-6)
	// Single-sample data at 20°C must not be included (HAVING count >= 2)
	require.NotContains(t, res[0], 20)
	// Old data from day -400 (temp 10°C) must not be included
	require.NotContains(t, res[0], 10)

	// Verify with percentile > 0
	settings.SetFloat(keys.ProfilePercentile, 50)
	t.Cleanup(func() { require.NoError(t, settings.Delete(keys.ProfilePercentile)) })

	resPerc, err := energyProfileTemperatureBinned(e)
	require.NoError(t, err)
	require.Len(t, resPerc, 96)
	require.Contains(t, resPerc[0], 4)
	require.InDelta(t, 0.55, resPerc[0][4], 1e-6)
}
