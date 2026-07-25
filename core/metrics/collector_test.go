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

	col, err := NewCollector("foo", "foo", "", WithClock(clock))
	require.NoError(t, err)
	require.True(t, col.accu.updated.IsZero())

	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))
	require.False(t, col.accu.updated.IsZero())

	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))
	require.Equal(t, 1e3*5/60/1e3, col.accu.Energy) // kWh

	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))
	require.Equal(t, 0.0, col.accu.Energy) // accumulator reset after 15 minutes

	clock.Add(15 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))
	require.Equal(t, 0.0, col.accu.Energy) // accumulator reset after 15 minutes
}

func TestCollectorAddEnergyWithEnergyMeter(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("bar", "bar", "", WithClock(clock))
	require.NoError(t, err)

	// first call: seeds meter, no delta yet
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(new(50000.0), nil, 1e3))
	require.Equal(t, 0.0, col.accu.Energy)

	// second call: meter delta of 0.5 kWh, power ignored for energy
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(new(50000.5), nil, 1e3))
	require.Equal(t, 0.5, col.accu.Energy)

	// implausible reading (decreased): ignored by guard
	clock.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(new(49000.0), nil, 1e3))
	require.Equal(t, 0.0, col.accu.Energy) // reset at slot boundary
}

func TestCollectorAddEnergyWithEnergyMeterAndReturn(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("baz", "baz", "", WithClock(clock))
	require.NoError(t, err)

	// seed energy meter
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(1000.0), nil, 0))

	// positive power: energy via meter delta, no return energy
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(1000.3), nil, 500))
	require.InDelta(t, 0.3, col.accu.Energy, 1e-10)
	require.Equal(t, 0.0, col.accu.ReturnEnergy)

	// negative power: energy via meter (no change), return energy via power integration
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(1000.3), nil, -600))
	require.InDelta(t, 0.3, col.accu.Energy, 1e-10)
	require.InDelta(t, 600.0*3/60/1e3, col.accu.ReturnEnergy, 1e-10) // 0.03 kWh
}

func TestCollectorAddEnergyWithReturnEnergyMeterAndEnergy(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("baz2", "baz2", "", WithClock(clock))
	require.NoError(t, err)

	// seed return energy meter
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, new(1000.0), 0))

	// negative power: return energy via meter delta, no energy
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, new(1000.3), -500))
	require.InDelta(t, 0.3, col.accu.ReturnEnergy, 1e-10)
	require.Equal(t, 0.0, col.accu.Energy)

	// positive power: return energy via meter (no change), energy via power integration
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, new(1000.3), 600))
	require.InDelta(t, 0.3, col.accu.ReturnEnergy, 1e-10)
	require.InDelta(t, 600.0*3/60/1e3, col.accu.Energy, 1e-10) // 0.03 kWh
}

func TestCollectorAddEnergyWithBothMeters(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("qux", "qux", "", WithClock(clock))
	require.NoError(t, err)

	// seed both meters
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(1000.0), new(2000.0), 0))

	// both deltas used, power ignored
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(1000.3), new(2000.7), 999))
	require.InDelta(t, 0.3, col.accu.Energy, 1e-10)
	require.InDelta(t, 0.7, col.accu.ReturnEnergy, 1e-10)
}

func TestCollectorSetEnergyAndReturnEnergyMeterTotal(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("set", "set", "", WithClock(clock))
	require.NoError(t, err)

	// seed both energy and return energy
	clock.Add(5 * time.Minute)
	require.NoError(t, col.SetEnergyMeterTotal(1000))
	require.NoError(t, col.SetReturnEnergyMeterTotal(2000))

	// both deltas used
	clock.Add(5 * time.Minute)
	require.NoError(t, col.SetEnergyMeterTotal(1000.3))
	require.NoError(t, col.SetReturnEnergyMeterTotal(2000.7))
	require.InDelta(t, 0.3, col.accu.Energy, 1e-10)
	require.InDelta(t, 0.7, col.accu.ReturnEnergy, 1e-10)
}

// TestCollectorRecoversAfterFullMeterFailure verifies that a meter whose power
// AND energy reads both fail for one cycle (collectMeters then passes power 0
// and a nil energy total) recovers cleanly: the meter delta on the next good
// read covers the gap and nothing is double-counted.
func TestCollectorRecoversAfterFullMeterFailure(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("full-fail", "full-fail", "", WithClock(clock))
	require.NoError(t, err)

	// seed energy meter at 10 kWh, constant 1 kW
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(10.0), nil, 1e3))
	require.Equal(t, 0.0, col.accu.Energy)

	// good read: meter advanced 0.05 kWh (3 min @ 1 kW)
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(10.05), nil, 1e3))
	require.InDelta(t, 0.05, col.accu.Energy, 1e-9)

	// both reads fail: power 0, energy nil -> AddPower(0) contributes nothing
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 0))

	// recovered read: meter advanced to 10.15 (real 0.10 kWh since last good read)
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(10.15), nil, 1e3))

	// 9 min @ 1 kW = 0.15 kWh delivered; meter 10.00 -> 10.15 = 0.15 kWh
	require.InDelta(t, 0.15, col.accu.Energy, 1e-9)
}

// TestCollectorRecoversAfterFailedEnergyRead verifies energy accounting when a
// meter's energy total read fails for one cycle while power keeps reporting
// (collectMeters passes a nil energy total but a valid power). A metered
// direction skips power integration on the failed read and recovers the gap via
// the next meter delta, so the failed interval is not double-counted.
func TestCollectorRecoversAfterFailedEnergyRead(t *testing.T) {
	clock := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("energy-fail", "energy-fail", "", WithClock(clock))
	require.NoError(t, err)

	// seed energy meter at 10 kWh, constant 1 kW
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(10.0), nil, 1e3))
	require.Equal(t, 0.0, col.accu.Energy)

	// good read: meter advanced 0.05 kWh (3 min @ 1 kW)
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(10.05), nil, 1e3))
	require.InDelta(t, 0.05, col.accu.Energy, 1e-9)

	// energy read fails (nil) while power is still reported: the gap is not
	// power-integrated because the meter has reported a total before
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))
	require.InDelta(t, 0.05, col.accu.Energy, 1e-9)

	// recovered read: meter advanced to 10.15 (real 0.10 kWh since last good read)
	clock.Add(3 * time.Minute)
	require.NoError(t, col.AddEnergy(new(10.15), nil, 1e3))

	// 9 min @ 1 kW = 0.15 kWh delivered (meter 10.00 -> 10.15). The recovery
	// delta (10.15 - 10.05) captures the gap exactly, without double-counting.
	require.InDelta(t, 0.15, col.accu.Energy, 1e-9)
}

// TestCollectorSkipsPartialFirstSlot verifies that the first slot, joined
// mid-way after (re)start, is not persisted as a full 15min slot.
func TestCollectorSkipsPartialFirstSlot(t *testing.T) {
	clk := clock.NewMock() // 1970-01-01 00:00:00 UTC

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector("partial", "partial", "", WithClock(clk))
	require.NoError(t, err)

	// first update mid-slot (00:05) - slot 00:00 is only partially covered
	clk.Add(5 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))
	clk.Add(5 * time.Minute) // 00:10
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))

	// cross into slot 00:15 - the partial slot 00:00 must not be persisted
	clk.Add(5 * time.Minute) // 00:15
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))

	var count int64
	require.NoError(t, db.Instance.Model(new(meter)).Count(&count).Error)
	require.Zero(t, count, "partial first slot must not be persisted")

	// cross into slot 00:30 - the fully covered slot 00:15 must be persisted
	clk.Add(15 * time.Minute) // 00:30
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))

	require.NoError(t, db.Instance.Model(new(meter)).Count(&count).Error)
	require.EqualValues(t, 1, count, "first full slot must be persisted")

	var m meter
	require.NoError(t, db.Instance.First(&m).Error)
	require.Equal(t, int64(15*60), m.Timestamp, "persisted slot should start at 00:15")
}

// TestCollectorRecoversDowntimeViaMeterReadings verifies that saved readings
// seed a new collector so the first slot after restart contains downtime energy.
func TestCollectorRecoversDowntimeViaMeterReadings(t *testing.T) {
	clk := clock.NewMock() // 1970-01-01 00:00:00 UTC, on a slot boundary

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector(Grid, "restore", "", WithClock(clk))
	require.NoError(t, err)
	require.False(t, col.restored, "fresh entity must not restore")

	// seed meters at slot start, advance within slot
	require.NoError(t, col.AddEnergy(new(100.0), new(200.0), 0))
	clk.Add(15 * time.Minute) // 00:15
	require.NoError(t, col.AddEnergy(new(100.5), new(200.2), 0))

	// cross into 00:30: slot 00:15 persisted, readings saved on entity
	clk.Add(15 * time.Minute) // 00:30
	require.NoError(t, col.AddEnergy(new(101.0), new(200.4), 0))

	var e entity
	require.NoError(t, db.Instance.First(&e, col.entity.Id).Error)
	require.Equal(t, 101.0, *e.EnergyMeter)
	require.Equal(t, 200.4, *e.ReturnEnergyMeter)

	var count int64
	require.NoError(t, db.Instance.Model(new(meter)).Where("meter = ?", col.entity.Id).Count(&count).Error)
	require.EqualValues(t, 2, count)

	// restart after 1h downtime, joining slot 01:30 mid-way
	clk.Add(65 * time.Minute) // 01:35
	col2, err := NewCollector(Grid, "restore", "", WithClock(clk))
	require.NoError(t, err)
	require.True(t, col2.restored)

	// first reading yields the delta across the downtime
	require.NoError(t, col2.AddEnergy(new(103.0), new(201.4), 0))
	require.InDelta(t, 2.0, col2.accu.Energy, 1e-10)
	require.InDelta(t, 1.0, col2.accu.ReturnEnergy, 1e-10)

	// cross into 01:45: catchup slot 01:30 persisted despite mid-slot start,
	// downtime delta plus the 0.5 kWh accrued since restart
	clk.Add(10 * time.Minute) // 01:45
	require.NoError(t, col2.AddEnergy(new(103.5), new(201.4), 0))

	var m meter
	require.NoError(t, db.Instance.Where("meter = ? AND ts = ?", col2.entity.Id, 90*60).First(&m).Error)
	require.InDelta(t, 2.5, m.Energy, 1e-10)
	require.InDelta(t, 1.0, m.ReturnEnergy, 1e-10)
	require.True(t, m.Recovered, "catchup slot must be flagged recovered")

	// the recovered slot is excluded from the household profile
	require.False(t, col2.restored, "recovery flag cleared after first slot")
	var recovered int64
	require.NoError(t, db.Instance.Model(new(meter)).Where("meter = ? AND recovered", col2.entity.Id).Count(&recovered).Error)
	require.EqualValues(t, 1, recovered, "only the catchup slot is recovered")

	// readings advanced with the persisted slot
	require.NoError(t, db.Instance.First(&e, col2.entity.Id).Error)
	require.Equal(t, 103.5, *e.EnergyMeter)
	require.Equal(t, 201.4, *e.ReturnEnergyMeter)
}

// TestCollectorRecoveryWithinCurrentSlot verifies that a restart that stays
// within the slot after the last persisted one keeps its meter delta
// time-correct: the catchup slot is persisted normally and not flagged
// recovered, so it stays in the household profile.
func TestCollectorRecoveryWithinCurrentSlot(t *testing.T) {
	clk := clock.NewMock() // 1970-01-01 00:00:00 UTC, on a slot boundary

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector(Grid, "within", "", WithClock(clk))
	require.NoError(t, err)

	// seed meters, advance a slot, cross into 00:30 to persist slot 00:15
	require.NoError(t, col.AddEnergy(new(100.0), new(200.0), 0))
	clk.Add(15 * time.Minute) // 00:15
	require.NoError(t, col.AddEnergy(new(100.5), new(200.2), 0))
	clk.Add(15 * time.Minute) // 00:30
	require.NoError(t, col.AddEnergy(new(101.0), new(200.4), 0))

	// restart 10min into slot 00:30 - the slot right after the last persisted
	clk.Add(10 * time.Minute) // 00:40
	col2, err := NewCollector(Grid, "within", "", WithClock(clk))
	require.NoError(t, err)
	require.True(t, col2.restored)
	require.EqualValues(t, 15*60, col2.lastSlot.Unix(), "last persisted slot is 00:15")

	// recovery happens during the current slot: the meter delta is applied but
	// no boundary was skipped, so nothing is persisted yet
	require.NoError(t, col2.AddEnergy(new(103.0), new(201.4), 0))
	require.InDelta(t, 2.0, col2.accu.Energy, 1e-10)
	var count int64
	require.NoError(t, db.Instance.Model(new(meter)).Where("meter = ?", col2.entity.Id).Count(&count).Error)
	require.EqualValues(t, 2, count, "current slot not persisted mid-slot")

	// cross into 00:45: slot 00:30 persisted with the full slot delta, not recovered
	clk.Add(5 * time.Minute) // 00:45
	require.NoError(t, col2.AddEnergy(new(103.5), new(201.4), 0))

	var m meter
	require.NoError(t, db.Instance.Where("meter = ? AND ts = ?", col2.entity.Id, 30*60).First(&m).Error)
	require.InDelta(t, 2.5, m.Energy, 1e-10)
	require.False(t, m.Recovered, "contiguous restart with meter totals is not recovered")

	// no slot is excluded from the household profile
	var recovered int64
	require.NoError(t, db.Instance.Model(new(meter)).Where("meter = ? AND recovered", col2.entity.Id).Count(&recovered).Error)
	require.Zero(t, recovered)
}

// TestCreateEntityRefreshesTitle verifies that a second call to createEntity
// with a non-empty title fills in (or updates) the title on an existing row,
// and that passing an empty title never clears a previously stored value.
func TestCreateEntityRefreshesTitle(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	// existing row with empty title (simulates pre-upgrade data)
	first, err := createEntity("grid", "grid", "")
	require.NoError(t, err)
	require.Empty(t, first.Title)

	// lazy-create with a real title must persist it
	second, err := createEntity("grid", "grid", "House meter")
	require.NoError(t, err)
	require.Equal(t, first.Id, second.Id, "should be the same row")
	require.Equal(t, "House meter", second.Title)

	var stored entity
	require.NoError(t, db.Instance.First(&stored, first.Id).Error)
	require.Equal(t, "House meter", stored.Title, "title must be persisted")

	// subsequent call with empty title must not clear the existing one
	third, err := createEntity("grid", "grid", "")
	require.NoError(t, err)
	require.Equal(t, "House meter", third.Title)

	require.NoError(t, db.Instance.First(&stored, first.Id).Error)
	require.Equal(t, "House meter", stored.Title, "title must survive empty re-create")

	// changing the title overwrites the stored value
	fourth, err := createEntity("grid", "grid", "Grid")
	require.NoError(t, err)
	require.Equal(t, "Grid", fourth.Title)

	require.NoError(t, db.Instance.First(&stored, first.Id).Error)
	require.Equal(t, "Grid", stored.Title)

	// only one row exists despite four createEntity calls with varying titles
	var count int64
	require.NoError(t, db.Instance.Model(new(entity)).Where("\"group\" = ? AND name = ?", "grid", "grid").Count(&count).Error)
	require.EqualValues(t, 1, count, "must not duplicate existing rows")
}

func TestCollectorSetSocTemp(t *testing.T) {
	clk := clock.NewMock() // 1970-01-01 00:00:00 UTC, on a slot boundary

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector(Battery, "bat", "", WithClock(clk))
	require.NoError(t, err)

	require.NoError(t, col.AddEnergy(nil, nil, 0))
	require.NoError(t, col.SetSocTemp(50, false))
	require.NoError(t, col.SetSocTemp(60, false)) // first reading per slot wins
	require.Equal(t, 50.0, *col.accu.SocTemp)

	// cross into the next slot: prior slot persisted, snapshot cleared
	clk.Add(15 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 0))
	require.Nil(t, col.accu.SocTemp)

	var m meter
	require.NoError(t, db.Instance.Where("meter = ?", col.entity.Id).First(&m).Error)
	require.Equal(t, 50.0, *m.SocTemp)

	// battery never marks is_temp, so the value is soc
	require.False(t, col.entity.IsTemp)

	// fresh slot captures its own start value
	require.NoError(t, col.SetSocTemp(70, false))
	require.Equal(t, 70.0, *col.accu.SocTemp)
}

func TestCollectorSetSocTempHeating(t *testing.T) {
	clk := clock.NewMock()

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector(Loadpoint, "lp", "", WithClock(clk))
	require.NoError(t, err)

	require.NoError(t, col.AddEnergy(nil, nil, 0))
	require.NoError(t, col.SetSocTemp(21.5, true)) // heating charger: value is temperature
	require.Equal(t, 21.5, *col.accu.SocTemp)

	clk.Add(15 * time.Minute)
	require.NoError(t, col.AddEnergy(nil, nil, 0))

	var m meter
	require.NoError(t, db.Instance.Where("meter = ?", col.entity.Id).First(&m).Error)
	require.Equal(t, 21.5, *m.SocTemp)

	// is_temp persisted on the entity row
	var e entity
	require.NoError(t, db.Instance.Where("id = ?", col.entity.Id).First(&e).Error)
	require.True(t, e.IsTemp)
}

// a meter regrouped to consumer keeps its id and history via in-place relabel
func TestCreateEntityReconcilesExtToConsumer(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	// ext meter with a persisted history slot
	ext, err := createEntity(Meter, "db:5", "Fridge")
	require.NoError(t, err)
	require.NoError(t, persist(ext, time.Unix(15*60, 0), 0.3, 0, nil, false))

	// reconfigured as consumer: same row relabeled, history intact
	con, err := createEntity(Consumer, "db:5", "Fridge")
	require.NoError(t, err)
	require.Equal(t, ext.Id, con.Id, "must reuse the existing row")
	require.Equal(t, Consumer, con.Group)

	var stored entity
	require.NoError(t, db.Instance.First(&stored, ext.Id).Error)
	require.Equal(t, Consumer, stored.Group, "group must be persisted")

	// no duplicate entity for the name
	var entities int64
	require.NoError(t, db.Instance.Model(new(entity)).Where("name = ?", "db:5").Count(&entities).Error)
	require.EqualValues(t, 1, entities)

	// history row still attached to the (now consumer) entity
	var meters int64
	require.NoError(t, db.Instance.Model(new(meter)).Where("meter = ?", ext.Id).Count(&meters).Error)
	require.EqualValues(t, 1, meters)
}

// an existing consumer row blocks the relabel, leaving the meter row untouched
func TestCreateEntityReconcileGuard(t *testing.T) {
	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	meterRow, err := createEntity(Meter, "db:7", "")
	require.NoError(t, err)

	// pre-existing consumer row blocks the in-place relabel
	conRow := entity{Group: Consumer, Name: "db:7"}
	require.NoError(t, db.Instance.Create(&conRow).Error)

	got, err := createEntity(Consumer, "db:7", "")
	require.NoError(t, err)
	require.Equal(t, conRow.Id, got.Id, "must reuse existing consumer row")

	// meter row untouched
	var stored entity
	require.NoError(t, db.Instance.First(&stored, meterRow.Id).Error)
	require.Equal(t, Meter, stored.Group)
}

func TestCollectorLastSlotEnergy(t *testing.T) {
	clk := clock.NewMock() // 1970-01-01 00:00:00 UTC, on a slot boundary

	require.NoError(t, db.NewInstance("sqlite", ":memory:"))
	require.NoError(t, SetupSchema())

	col, err := NewCollector(Home, "last", "", WithClock(clk))
	require.NoError(t, err)

	// nothing persisted yet
	_, ok := col.LastSlotEnergy()
	require.False(t, ok)

	// constant 1 kW: crossing into 00:30 persists slot 00:15
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))
	clk.Add(15 * time.Minute) // 00:15
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))
	clk.Add(15 * time.Minute) // 00:30
	require.NoError(t, col.AddEnergy(nil, nil, 1e3))

	v, ok := col.LastSlotEnergy()
	require.True(t, ok)
	require.InDelta(t, 0.25, v, 1e-10) // kWh

	// mid-slot the previous slot stays the reference
	clk.Add(5 * time.Minute) // 00:35
	v, ok = col.LastSlotEnergy()
	require.True(t, ok)
	require.InDelta(t, 0.25, v, 1e-10)

	// a recovered slot is excluded
	require.NoError(t, db.Instance.Model(new(meter)).Where("meter = ? AND ts = ?", col.entity.Id, 15*60).Update("recovered", true).Error)
	_, ok = col.LastSlotEnergy()
	require.False(t, ok)
}
