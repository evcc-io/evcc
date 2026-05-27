package metrics

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/require"
)

func TestListEntities(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	// SetupSchema reserves the home entity (id 1) without data
	grid := entity{Id: 2, Name: "grid", Group: Grid}
	require.NoError(t, db.Instance.Create(&grid).Error)
	pv := entity{Id: 3, Name: "pv1", Group: PV}
	require.NoError(t, db.Instance.Create(&pv).Error)

	base := time.Date(2026, 4, 15, 16, 0, 0, 0, time.Now().Location())
	require.NoError(t, persist(grid, base, 1, 0))
	require.NoError(t, persist(grid, base.Add(time.Hour), 2, 0))

	entities, err := ListEntities()
	require.NoError(t, err)
	require.Len(t, entities, 3) // home + grid + pv1

	byName := make(map[string]EntityInfo, len(entities))
	for _, e := range entities {
		byName[e.Name] = e
	}

	// entity with data: slot count and range
	require.Equal(t, Grid, byName["grid"].Group)
	require.Equal(t, 2, byName["grid"].Slots)
	require.True(t, base.Equal(byName["grid"].First))
	require.True(t, base.Add(time.Hour).Equal(byName["grid"].Last))

	// entity without data: zero slots, zero timestamps
	require.Equal(t, 0, byName["pv1"].Slots)
	require.True(t, byName["pv1"].First.IsZero())
	require.True(t, byName["pv1"].Last.IsZero())

	// home entity exists but has no data
	require.Equal(t, 0, byName["home"].Slots)
}
