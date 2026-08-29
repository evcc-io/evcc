package metrics

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/require"
)

func TestDeleteEnergy(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	ePv := entity{Id: 2, Name: "pv1", Group: PV}
	require.NoError(t, db.Instance.Create(&ePv).Error)
	eFc := entity{Id: 3, Name: Forecast, Group: Forecast}
	require.NoError(t, db.Instance.Create(&eFc).Error)

	loc := time.Now().Location()
	base := time.Date(2026, 4, 15, 16, 0, 0, 0, loc)
	for i := range 4 {
		ts := base.Add(time.Duration(i) * 15 * time.Minute)
		require.NoError(t, persist(ePv, ts, 1, 0, nil, false))
		require.NoError(t, persist(eFc, ts, 2, 0, nil, false))
	}

	count := func() int64 {
		var n int64
		require.NoError(t, db.Instance.Model(new(meter)).Count(&n).Error)
		return n
	}
	require.Equal(t, int64(8), count())

	// both bounds are required
	_, err := DeleteEnergy(time.Time{}, base, EnergyFilter{})
	require.Error(t, err)

	// filter narrows to the matching entity, range is half-open
	rows, err := DeleteEnergy(base.UTC(), base.Add(30*time.Minute).UTC(), EnergyFilter{Group: Forecast})
	require.NoError(t, err)
	require.Equal(t, int64(2), rows)
	require.Equal(t, int64(6), count())

	// pv untouched
	res, err := QueryEnergy(base.Add(-time.Hour).UTC(), base.Add(time.Hour).UTC(), "hour", false, EnergyFilter{Group: PV})
	require.NoError(t, err)
	require.Len(t, res, 1)
	require.Equal(t, 4.0, res[0].Data[0].Energy)

	// empty filter deletes across entities
	rows, err = DeleteEnergy(base.UTC(), base.Add(time.Hour).UTC(), EnergyFilter{})
	require.NoError(t, err)
	require.Equal(t, int64(6), rows)
	require.Zero(t, count())
}

func TestDeleteTariffs(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, db.Instance.AutoMigrate(new(tariffValue)))

	base := time.Date(2026, 4, 15, 16, 0, 0, 0, time.UTC)
	grid, co2 := 0.3, 250.0
	for i := range 4 {
		require.NoError(t, PersistTariffs(base.Add(time.Duration(i)*15*time.Minute), &grid, nil, &co2, nil))
	}

	count := func() int64 {
		var n int64
		require.NoError(t, db.Instance.Model(new(tariffValue)).Count(&n).Error)
		return n
	}

	_, err := DeleteTariffs(base, time.Time{}, "")
	require.Error(t, err)

	// unknown usage never reaches the column interpolation
	_, err = DeleteTariffs(base, base.Add(time.Hour), "unknown")
	require.ErrorIs(t, err, ErrInvalidUsage)
	require.Equal(t, int64(4), count())

	// clearing one usage keeps the row as long as another value remains
	rows, err := DeleteTariffs(base, base.Add(30*time.Minute), "grid")
	require.NoError(t, err)
	require.Equal(t, int64(2), rows)
	require.Equal(t, int64(4), count())

	var res tariffValue
	require.NoError(t, db.Instance.Where("ts = ?", base.Unix()).First(&res).Error)
	require.Nil(t, res.Grid)
	require.InDelta(t, 250, *res.Co2, 0.001)

	// clearing the last usage drops the now empty rows
	rows, err = DeleteTariffs(base, base.Add(30*time.Minute), "co2")
	require.NoError(t, err)
	require.Equal(t, int64(2), rows)
	require.Equal(t, int64(2), count())

	// no usage drops the whole row
	rows, err = DeleteTariffs(base, base.Add(time.Hour), "")
	require.NoError(t, err)
	require.Equal(t, int64(2), rows)
	require.Zero(t, count())
}
