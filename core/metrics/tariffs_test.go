package metrics

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/db"
	"github.com/stretchr/testify/require"
)

func TestPersistTariffs(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, db.Instance.AutoMigrate(new(tariffValue)))

	slot := time.Date(2026, 4, 15, 16, 15, 0, 0, time.UTC)
	grid, co2 := 0.3, 250.0

	// nil values omitted
	require.NoError(t, PersistTariffs(slot, &grid, nil, &co2, nil))

	var res tariffValue
	require.NoError(t, db.Instance.First(&res).Error)
	require.Equal(t, slot.Unix(), res.Timestamp)
	require.InDelta(t, 0.3, *res.Grid, 0.001)
	require.Nil(t, res.FeedIn)
	require.InDelta(t, 250, *res.Co2, 0.001)
	require.Nil(t, res.Temperature)

	// duplicate slot ignored, first values kept
	other := 0.4
	require.NoError(t, PersistTariffs(slot, &other, nil, nil, nil))

	var count int64
	require.NoError(t, db.Instance.Model(new(tariffValue)).Count(&count).Error)
	require.Equal(t, int64(1), count)

	require.NoError(t, db.Instance.First(&res).Error)
	require.InDelta(t, 0.3, *res.Grid, 0.001)

	// all nil: no row
	require.NoError(t, PersistTariffs(slot.Add(15*time.Minute), nil, nil, nil, nil))
	require.NoError(t, db.Instance.Model(new(tariffValue)).Count(&count).Error)
	require.Equal(t, int64(1), count)

	// all values set: each column mapped independently
	next := slot.Add(30 * time.Minute)
	feedin, temp := 0.08, 21.5
	require.NoError(t, PersistTariffs(next, &grid, &feedin, &co2, &temp))

	require.NoError(t, db.Instance.Where("ts = ?", next.Unix()).First(&res).Error)
	require.InDelta(t, 0.3, *res.Grid, 0.001)
	require.InDelta(t, 0.08, *res.FeedIn, 0.001)
	require.InDelta(t, 250, *res.Co2, 0.001)
	require.InDelta(t, 21.5, *res.Temperature, 0.001)
}

func TestQueryTariffs(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, db.Instance.AutoMigrate(new(tariffValue)))

	slot := time.Date(2026, 4, 15, 16, 0, 0, 0, time.UTC)
	grid := 0.3
	for i := range 3 {
		v := grid + float64(i)/10
		require.NoError(t, PersistTariffs(slot.Add(time.Duration(i)*15*time.Minute), &v, nil, nil, nil))
	}

	res, err := QueryTariffs(slot.Add(15*time.Minute), slot.Add(45*time.Minute), "")
	require.NoError(t, err)
	require.Len(t, res, 2)
	require.Equal(t, slot.Add(15*time.Minute).Unix(), res[0].Start.Unix())
	require.InDelta(t, 0.4, *res[0].Grid, 0.001)
	require.Nil(t, res[0].FeedIn)

	// one hour bucket: average with the slot range
	agg, err := QueryTariffs(slot, slot.Add(time.Hour), "hour")
	require.NoError(t, err)
	require.Len(t, agg, 1)
	require.InDelta(t, 0.4, *agg[0].Grid, 0.001)
	require.InDelta(t, 0.3, *agg[0].GridMin, 0.001)
	require.InDelta(t, 0.5, *agg[0].GridMax, 0.001)
}
