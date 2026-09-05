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
	res, err := energyProfileFiltered(e, now.BeginningOfDay().AddDate(0, 0, -28), &weekday)
	require.NoError(t, err)

	// only same-weekday slots must be averaged
	for i, v := range res {
		require.Equal(t, 2.0, v, "slot %d", i)
	}
}
