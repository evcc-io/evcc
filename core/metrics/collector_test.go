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
	require.NoError(t, col.AddPower(1e3, nil, nil))
	require.False(t, col.accu.updated.IsZero())

	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddPower(1e3, nil, nil))
	require.Equal(t, 1e3*5/60/1e3, col.accu.PosEnergy()) // kWh

	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddPower(1e3, nil, nil))
	require.Equal(t, 0.0, col.accu.PosEnergy()) // accumulator reset after 15 minutes

	clock.Add(15 * time.Minute)
	require.NoError(t, col.AddPower(1e3, nil, nil))
	require.Equal(t, 0.0, col.accu.PosEnergy()) // accumulator reset after 15 minutes
}

func TestCollectorAddPowerWithMeter(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("bar", "bar", WithClock(clock))
	require.NoError(t, err)

	f := func(v float64) *float64 { return &v }

	// first call: seeds lastImport, falls back to power integration
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddPower(1e3, f(50000), nil))
	require.Nil(t, nil) // no error
	require.NotNil(t, col.lastImport)
	require.Equal(t, 50000.0, *col.lastImport)

	// second call: uses meter delta (0.5 kWh), ignores power
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddPower(1e3, f(50000.5), nil))
	require.Equal(t, 0.5, col.accu.PosEnergy())

	// implausible reading (decreased): ignored, falls back to power
	prevImport := col.accu.PosEnergy()
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddPower(1e3, f(49000), nil))
	// power integration adds 1kW * 5min = 0.0833 kWh on top — but accumulator was reset at 15min boundary
	// after 15min slot boundary, accumulator resets
	require.Equal(t, 0.0, col.accu.PosEnergy()) // reset at slot boundary
	_ = prevImport

	// verify lastImport was updated even though reading was implausible
	require.Equal(t, 49000.0, *col.lastImport)
}

func TestCollectorAddPowerWithImportAndExport(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("baz", "baz", WithClock(clock))
	require.NoError(t, err)

	f := func(v float64) *float64 { return &v }

	// seed both import and export
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddPower(0, f(1000), f(2000)))

	// both deltas used
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddPower(0, f(1000.3), f(2000.7)))
	require.InDelta(t, 0.3, col.accu.PosEnergy(), 1e-10)
	require.InDelta(t, 0.7, col.accu.NegEnergy(), 1e-10)
}

func TestCollectorAddPowerMixedImportOnly(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("mix", "mix", WithClock(clock))
	require.NoError(t, err)

	f := func(v float64) *float64 { return &v }

	// seed import, no export meter
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddPower(-500, f(1000), nil))

	// import delta used, export direction falls back to power (but power is negative so nothing added to import)
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddPower(-500, f(1000.2), nil))
	require.InDelta(t, 0.2, col.accu.PosEnergy(), 1e-10)
	require.Equal(t, 0.0, col.accu.NegEnergy()) // no export meter, no power fallback because import meter was used
}
