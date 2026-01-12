package metrics

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/require"
)

func TestCollectorAddPower(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("foo", "foo", WithClock(clock))
	require.NoError(t, err)
	require.True(t, col.accu.updated.IsZero())

	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddPower(1e3))
	require.False(t, col.accu.updated.IsZero())

	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddPower(1e3))
	require.Equal(t, 1e3*5/60/1e3, col.accu.PosEnergy()) // kWh

	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddPower(1e3))
	require.Equal(t, 0.0, col.accu.PosEnergy()) // accumulator reset after 15 minutes

	clock.Add(15 * time.Minute)
	require.NoError(t, col.AddPower(1e3))
	require.Equal(t, 0.0, col.accu.PosEnergy()) // accumulator reset after 15 minutes
}
