package metrics

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/db"
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

func TestEnergyProfileActiveDays(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	e := entity{Id: 3, Name: "heater1", Group: Loadpoint}
	require.NoError(t, db.Instance.Create(&e).Error)

	// 14 past days:
	// days -14..-8: 7 active days with energy = 0.1 kWh/slot (total 9.6 kWh/day >= 5.0 kWh threshold)
	// days -7..-1:  7 warm/idle days with energy = 0.001 kWh/slot (total 0.096 kWh/day < 5.0 kWh threshold)
	for day := -14; day < 0; day++ {
		base := now.BeginningOfDay().AddDate(0, 0, day)
		energy := 0.001
		if day < -7 {
			energy = 0.1
		}

		for slot := range 96 {
			ts := base.Add(time.Duration(slot) * tariff.SlotDuration)
			require.NoError(t, persist(e, ts, energy, 0, nil, false))
		}
	}

	// Active days profile should skip the 7 warm days and average the 7 active days (0.1 kWh/slot)
	res, err := energyProfileActiveDays(e, 7, 5.0, 0)
	require.NoError(t, err)

	for i, v := range res {
		require.InDelta(t, 0.1, v, 1e-6, "slot %d", i)
	}

	// If threshold is higher than any day (e.g. 50 kWh), ErrIncomplete should be returned
	_, err = energyProfileActiveDays(e, 7, 50.0, 0)
	require.ErrorIs(t, err, ErrIncomplete)
}
