package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/jinzhu/now"
	"github.com/stretchr/testify/assert"
)

func TestMeterEnergy(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &meterEnergy{
		clock:     clock,
		startFunc: beginningOfDay,
	}

	me.AddMeterTotal(10)
	assert.Equal(t, 0.0, me.AccumulatedEnergy())
	me.AddMeterTotal(11)
	assert.Equal(t, 1.0, me.AccumulatedEnergy())
	me.AddMeterTotal(11)
	assert.Equal(t, 1.0, me.AccumulatedEnergy())

	me.AddPower(1e3)
	assert.Equal(t, 1.0, me.AccumulatedEnergy())

	clock.Add(30 * time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 1.5, me.AccumulatedEnergy())

	clock.Add(23*time.Hour + 30*time.Minute)
	me.AddPower(1e3)
	assert.Equal(t, 0.0, me.AccumulatedEnergy())

	clock.Add(1 * time.Hour)
	me.AddPower(1e3)
	assert.Equal(t, 1.0, me.AccumulatedEnergy())

	me.AddMeterTotal(12)
	assert.Equal(t, 1.0, me.AccumulatedEnergy())
	me.AddMeterTotal(13)
	assert.Equal(t, 2.0, me.AccumulatedEnergy())
}
