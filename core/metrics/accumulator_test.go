package metrics

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
)

func TestMeterEnergyMeterTotal(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &Accumulator{clock: clock}

	me.SetImportMeterTotal(10)
	assert.Equal(t, 0.0, me.Imported())
	me.SetImportMeterTotal(11)
	assert.Equal(t, 1.0, me.Imported())
	me.SetImportMeterTotal(11)
	assert.Equal(t, 1.0, me.Imported())
}

// TestMeterEnergyMeterTotalIntermittentZero covers a grid meter that briefly
// reports 0 (e.g. an inverter timeout/restart) before recovering to its real
// cumulative value. The spurious drop must be ignored so the recovery is not
// counted as a spike. https://github.com/evcc-io/evcc/discussions/30555
func TestMeterEnergyMeterTotalIntermittentZero(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &Accumulator{clock: clock}

	me.SetImportMeterTotal(13422.7)
	assert.Equal(t, 0.0, me.Imported())

	clock.Add(15 * time.Minute)
	me.SetImportMeterTotal(13423.7)
	assert.Equal(t, 1.0, me.Imported())

	// meter drops out and reports 0 for one slot
	clock.Add(15 * time.Minute)
	me.SetImportMeterTotal(0)
	assert.Equal(t, 1.0, me.Imported(), "spurious zero must not change energy")

	// meter recovers: difference is taken against the last valid value, not 0
	clock.Add(15 * time.Minute)
	me.SetImportMeterTotal(13424.7)
	assert.Equal(t, 2.0, me.Imported(), "recovery must not produce a spike")
}

// TestMeterEnergyExportMeterTotalIntermittentZero mirrors the import case for
// the export meter total.
func TestMeterEnergyExportMeterTotalIntermittentZero(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &Accumulator{clock: clock}

	me.SetExportMeterTotal(500)
	assert.Equal(t, 0.0, me.Exported())

	me.SetExportMeterTotal(501)
	assert.Equal(t, 1.0, me.Exported())

	// spurious drop to 0 must be ignored, not rebased
	me.SetExportMeterTotal(0)
	assert.Equal(t, 1.0, me.Exported())

	me.SetExportMeterTotal(502)
	assert.Equal(t, 2.0, me.Exported())
}

// TestMeterEnergyAddPowerIntermittentZero verifies the power-integration path
// is robust to a missing/zero intermediate reading: it simply adds no energy
// for that interval rather than corrupting the accumulated total.
func TestMeterEnergyAddPowerIntermittentZero(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &Accumulator{clock: clock}

	clock.Add(60 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 0.0, me.Imported())

	clock.Add(60 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 1.0, me.Imported())

	// power reads 0 (missing value): no energy added for this interval
	clock.Add(60 * time.Minute)
	me.AddPower(0)
	assert.Equal(t, 1.0, me.Imported())

	clock.Add(60 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 2.0, me.Imported())
}

func TestMeterEnergyAddPower(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &Accumulator{clock: clock}

	clock.Add(60 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 0.0, me.Imported())

	clock.Add(60 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 1.0, me.Imported())

	clock.Add(30 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 1.5, me.Imported())
}
