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
