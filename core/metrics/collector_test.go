package metrics

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/server/db"
	"github.com/stretchr/testify/require"
)

func TestCollectorAddEnergy(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("foo", "foo", WithClock(clock))
	require.NoError(t, err)
	require.True(t, col.accu.updated.IsZero())

	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))
	require.False(t, col.accu.updated.IsZero())

	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))
	require.Equal(t, 1e3*5/60/1e3, col.accu.Imported()) // kWh

	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))
	require.Equal(t, 0.0, col.accu.Imported()) // accumulator reset after 15 minutes

	clock.Add(15 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))
	require.Equal(t, 0.0, col.accu.Imported()) // accumulator reset after 15 minutes
}

func TestCollectorAddEnergyWithImportMeter(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("bar", "bar", WithClock(clock))
	require.NoError(t, err)

	// first call: seeds meter, no delta yet
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(new(50000.0), nil, 1e3))
	require.Equal(t, 0.0, col.accu.Imported())

	// second call: meter delta of 0.5 kWh, power ignored for import
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(new(50000.5), nil, 1e3))
	require.Equal(t, 0.5, col.accu.Imported())

	// implausible reading (decreased): ignored by guard
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(new(49000.0), nil, 1e3))
	require.Equal(t, 0.0, col.accu.Imported()) // reset at slot boundary
}

func TestCollectorAddEnergyWithImportMeterAndExport(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("baz", "baz", WithClock(clock))
	require.NoError(t, err)

	// seed import meter
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(1000.0), nil, 0))

	// positive power: import via meter delta, no export
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(1000.3), nil, 500))
	require.InDelta(t, 0.3, col.accu.Imported(), 1e-10)
	require.Equal(t, 0.0, col.accu.Exported())

	// negative power: import via meter (no change), export via power integration
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(1000.3), nil, -600))
	require.InDelta(t, 0.3, col.accu.Imported(), 1e-10)
	require.InDelta(t, 600.0*3/60/1e3, col.accu.Exported(), 1e-10) // 0.03 kWh
}

func TestCollectorAddEnergyWithExportMeterAndImport(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("baz2", "baz2", WithClock(clock))
	require.NoError(t, err)

	// seed export meter
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, new(1000.0), 0))

	// negative power: export via meter delta, no import
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, new(1000.3), -500))
	require.InDelta(t, 0.3, col.accu.Exported(), 1e-10)
	require.Equal(t, 0.0, col.accu.Imported())

	// positive power: export via meter (no change), import via power integration
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, new(1000.3), 600))
	require.InDelta(t, 0.3, col.accu.Exported(), 1e-10)
	require.InDelta(t, 600.0*3/60/1e3, col.accu.Imported(), 1e-10) // 0.03 kWh
}

func TestCollectorAddEnergyWithBothMeters(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("qux", "qux", WithClock(clock))
	require.NoError(t, err)

	// seed both meters
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(1000.0), new(2000.0), 0))

	// both deltas used, power ignored
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(1000.3), new(2000.7), 999))
	require.InDelta(t, 0.3, col.accu.Imported(), 1e-10)
	require.InDelta(t, 0.7, col.accu.Exported(), 1e-10)
}

func TestCollectorSetImportAndExportMeterTotal(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("set", "set", WithClock(clock))
	require.NoError(t, err)

	// seed both import and export
	clock.Add(5 * time.Minute)
	require.NoError(t, col.SetImportMeterTotal(1000))
	require.NoError(t, col.SetExportMeterTotal(2000))

	// both deltas used
	clock.Add(5 * time.Minute)
	require.NoError(t, col.SetImportMeterTotal(1000.3))
	require.NoError(t, col.SetExportMeterTotal(2000.7))
	require.InDelta(t, 0.3, col.accu.Imported(), 1e-10)
	require.InDelta(t, 0.7, col.accu.Exported(), 1e-10)
}
