package metrics

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFill(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	slot := func(n int) time.Time {
		return time.Unix(int64(n)*slotSeconds, 0)
	}

	e, err := createEntity(Grid, "fill", "")
	require.NoError(t, err)

	require.NoError(t, persist(e, slot(0), 1, 4, nil, false))
	// slot 1 missing
	require.NoError(t, persist(e, slot(2), 5, 0, nil, false))
	// slot 3 missing, slot 4 holds its energy: untouched
	require.NoError(t, persist(e, slot(4), 4, 2, nil, true))
	// slots 5-6 missing: downtime, stays empty
	require.NoError(t, persist(e, slot(7), 1, 0, nil, false))

	for range 2 { // idempotent
		for _, n := range []int{0, 2, 4, 7} {
			require.NoError(t, fill(e, slot(n)))
		}

		var rows []meter
		require.NoError(t, db.Instance.Where("meter = ?", e.Id).Order("ts").Find(&rows).Error)
		require.Len(t, rows, 5)

		for i, tc := range []struct {
			slot                 int
			energy, returnEnergy float64
			origin               int
		}{
			{0, 1, 4, slotMeasured},
			{1, 3, 2, slotInterpolated},
			{2, 5, 0, slotMeasured},
			{4, 4, 2, slotRecovered},
			{7, 1, 0, slotMeasured},
		} {
			assert.Equal(t, slot(tc.slot).Unix(), rows[i].Timestamp)
			assert.InDelta(t, tc.energy, rows[i].Energy, 1e-10, "slot %d", tc.slot)
			assert.InDelta(t, tc.returnEnergy, rows[i].ReturnEnergy, 1e-10, "slot %d", tc.slot)
			assert.Equal(t, tc.origin, rows[i].Recovered, "slot %d", tc.slot)
		}
	}
}

func TestBackfill(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	e, err := createEntity(Grid, "backfill", "")
	require.NoError(t, err)

	// existing history with slots 1 and 3 missing, backfill pending
	for _, n := range []int64{0, 2, 4} {
		require.NoError(t, persist(e, time.Unix(n*slotSeconds, 0), 1, 0, nil, false))
	}
	require.NoError(t, settings.Delete(keys.MetricsFillVersion))
	require.NoError(t, db.Instance.Create(&flowDay{Day: dayStart(time.Unix(86400, 0)).Unix(), Data: "{}"}).Error)

	require.NoError(t, SetupSchema())
	require.Empty(t, rolled(t), "rollup dropped")

	// both gaps filled, remove them again
	res := db.Instance.Where("meter = ? AND recovered = ?", e.Id, slotInterpolated).Delete(new(meter))
	require.NoError(t, res.Error)
	assert.Equal(t, int64(2), res.RowsAffected)

	// already backfilled: gaps stay
	require.NoError(t, SetupSchema())

	var rows int64
	require.NoError(t, db.Instance.Model(new(meter)).Where("meter = ?", e.Id).Count(&rows).Error)
	assert.Equal(t, int64(3), rows)
}
