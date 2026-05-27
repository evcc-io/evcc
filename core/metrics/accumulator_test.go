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

// A transient low/zero meter reading (Modbus glitch) must not become the new
// baseline. Regression test for evcc-io/evcc#29286.
func TestMeterEnergyMeterTotalRollback(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &Accumulator{clock: clock}

	me.SetImportMeterTotal(25567.546)
	assert.Equal(t, 0.0, me.Imported())
	me.SetImportMeterTotal(25567.548)
	assert.InDelta(t, 0.002, me.Imported(), 1e-9)

	good := me.Updated()
	clock.Add(time.Second)
	me.SetImportMeterTotal(0) // transient bad read
	assert.InDelta(t, 0.002, me.Imported(), 1e-9)
	assert.Equal(t, good, me.Updated(), "updated must not advance on rejected rollback")

	clock.Add(time.Second)
	me.SetImportMeterTotal(25567.550) // baseline preserved, delta is +0.002
	assert.InDelta(t, 0.004, me.Imported(), 1e-9)
	assert.True(t, me.Updated().After(good), "updated must advance on accepted reading")

	me.SetExportMeterTotal(100)
	me.SetExportMeterTotal(101)
	assert.Equal(t, 1.0, me.Exported())
	me.SetExportMeterTotal(0)
	assert.Equal(t, 1.0, me.Exported())
	me.SetExportMeterTotal(102)
	assert.Equal(t, 2.0, me.Exported())
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
