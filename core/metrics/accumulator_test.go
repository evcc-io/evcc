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

	me.SetEnergyMeterTotal(10)
	assert.Equal(t, 0.0, me.Energy)
	me.SetEnergyMeterTotal(11)
	assert.Equal(t, 1.0, me.Energy)
	me.SetEnergyMeterTotal(11)
	assert.Equal(t, 1.0, me.Energy)
}

func TestMeterEnergyMeterTotalDropout(t *testing.T) {
	me := &Accumulator{clock: clock.NewMock()}

	// transient lower reading must not rebase the baseline (#33820)
	me.SetEnergyMeterTotal(20658)
	me.SetEnergyMeterTotal(11179)
	me.SetEnergyMeterTotal(20659)
	assert.Equal(t, 1.0, me.Energy)

	// a repeated lower reading that does not increase is still not accepted
	me.SetEnergyMeterTotal(5)
	me.SetEnergyMeterTotal(5)
	me.SetEnergyMeterTotal(20660)
	assert.Equal(t, 2.0, me.Energy)

	// an increasing lower reading confirms a counter reset without adding energy
	me.SetEnergyMeterTotal(5)
	me.SetEnergyMeterTotal(6)
	assert.Equal(t, 2.0, me.Energy)
	me.SetEnergyMeterTotal(8)
	assert.Equal(t, 4.0, me.Energy)
}

func TestMeterEnergyMeterTotalSpike(t *testing.T) {
	me := &Accumulator{clock: clock.NewMock()}

	// single implausible reading is ignored (#35976)
	me.SetEnergyMeterTotal(1000)
	me.SetEnergyMeterTotal(1.8446744073709552e17)
	me.SetEnergyMeterTotal(1001)
	assert.Equal(t, 1.0, me.Energy)

	// confirmed jump is accepted, e.g. after downtime
	me.SetEnergyMeterTotal(1500)
	assert.Equal(t, 1.0, me.Energy)
	me.SetEnergyMeterTotal(1501)
	assert.Equal(t, 501.0, me.Energy)
}

func TestMeterEnergyAddPower(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &Accumulator{clock: clock}

	clock.Add(60 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 0.0, me.Energy)

	clock.Add(60 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 1.0, me.Energy)

	clock.Add(30 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 1.5, me.Energy)
}
