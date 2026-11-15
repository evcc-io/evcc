package metrics

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/db"
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
	// slots 1-3 missing
	require.NoError(t, persist(e, slot(4), 5, 0, nil, false))
	// slot 5 missing, slot 6 holds its energy
	require.NoError(t, persist(e, slot(6), 4, 2, nil, true))
	// slots 7-11 missing: downtime, stays empty
	require.NoError(t, persist(e, slot(12), 1, 0, nil, false))

	for range 2 { // idempotent
		require.NoError(t, fill(e))

		var rows []meter
		require.NoError(t, db.Instance.Where("meter = ?", e.Id).Order("ts").Find(&rows).Error)
		require.Len(t, rows, 8)

		for i, tc := range []struct {
			slot                 int
			energy, returnEnergy float64
			origin               int
		}{
			{0, 1, 4, slotMeasured},
			{1, 2, 3, slotInterpolated},
			{2, 3, 2, slotInterpolated},
			{3, 4, 1, slotInterpolated},
			{4, 5, 0, slotMeasured},
			{5, 2, 1, slotInterpolated},
			{6, 2, 1, slotRecovered},
			{12, 1, 0, slotMeasured},
		} {
			assert.Equal(t, slot(tc.slot).Unix(), rows[i].Timestamp)
			assert.InDelta(t, tc.energy, rows[i].Energy, 1e-10, "slot %d", tc.slot)
			assert.InDelta(t, tc.returnEnergy, rows[i].ReturnEnergy, 1e-10, "slot %d", tc.slot)
			assert.Equal(t, tc.origin, rows[i].Recovered, "slot %d", tc.slot)
		}
	}
}
