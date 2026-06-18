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

// TestMeterEnergyTransientDropout ensures a single bad/zero reading (e.g. a
// device sentinel decoded as 0) neither lowers the baseline nor injects a delta
// on recovery - see evcc-io/evcc#30950.
func TestMeterEnergyTransientDropout(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &Accumulator{clock: clock}

	me.SetEnergyMeterTotal(100)
	me.SetEnergyMeterTotal(110)
	assert.Equal(t, 10.0, me.Energy)

	// transient dropout to 0 - must be ignored
	me.SetEnergyMeterTotal(0)
	assert.Equal(t, 10.0, me.Energy)

	// recovery to the previous level - no bogus spike
	me.SetEnergyMeterTotal(110)
	assert.Equal(t, 10.0, me.Energy)

	me.SetEnergyMeterTotal(112)
	assert.Equal(t, 12.0, me.Energy)
}

// TestMeterEnergyTwoZeroReadings ensures two consecutive zero readings still do
// not establish 0 as a new baseline and spike on recovery.
func TestMeterEnergyTwoZeroReadings(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &Accumulator{clock: clock}

	me.SetEnergyMeterTotal(100)
	me.SetEnergyMeterTotal(0)
	me.SetEnergyMeterTotal(0)
	assert.Equal(t, 0.0, me.Energy)

	me.SetEnergyMeterTotal(105)
	assert.Equal(t, 5.0, me.Energy)
}

// TestMeterEnergyGenuineReset accepts a real counter reset once a second,
// increasing reading confirms the counter resumed from a low base.
func TestMeterEnergyGenuineReset(t *testing.T) {
	clock := clock.NewMock()
	clock.Set(now.BeginningOfDay())

	me := &Accumulator{clock: clock}

	me.SetEnergyMeterTotal(1000)
	me.SetEnergyMeterTotal(1010)
	assert.Equal(t, 10.0, me.Energy)

	// counter reset to a low base, then resumes counting
	me.SetEnergyMeterTotal(5)
	assert.Equal(t, 10.0, me.Energy) // unconfirmed, baseline kept
	me.SetEnergyMeterTotal(6)
	assert.Equal(t, 11.0, me.Energy) // confirmed: +1
	me.SetEnergyMeterTotal(9)
	assert.Equal(t, 14.0, me.Energy) // +3
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
