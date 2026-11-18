package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

// deadMeter is a power meter with nothing to say — a plug whose power is gone, a cloud that stopped
// answering. It reports api.ErrNotAvailable, which is what the homeassistant plugin does for an entity
// in state `unavailable`.
type deadMeter struct{ api.Meter }

func (deadMeter) CurrentPower() (float64, error) { return 0, api.ErrNotAvailable }

// TestChargePowerGoesDark: a failed reading must not leave the last wattage standing forever.
// Measured 2026-11-01: a Stiebel radiator showed 1807 W for 3,5 h while its plug had no power at all,
// because an error neither updated lp.chargePower nor the retained MQTT value.
func TestChargePowerGoesDark(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), settings.NewDatabaseSettingsAdapter("foo"))
	lp.chargeMeter = newChargeMeter(deadMeter{})
	lp.chargePower = 1807.4

	// a blip must not flicker the display
	for i := 0; i < chargePowerGrace-1; i++ {
		fails, dark := lp.chargePowerFailed()
		assert.False(t, dark, "no darkness before the grace period is over")
		assert.Equal(t, i+1, fails)
	}
	assert.Equal(t, 1807.4, lp.chargePower, "the last reading is kept during the grace period")

	// the tick that ends the grace period goes dark and erases the published value
	fails, dark := lp.chargePowerFailed()
	assert.Equal(t, chargePowerGrace, fails)
	assert.True(t, dark, "the loadpoint goes dark exactly once")
	assert.Equal(t, 0.0, lp.chargePower)

	// staying dark does not republish anything
	fails, dark = lp.chargePowerFailed()
	assert.False(t, dark)
	assert.Equal(t, chargePowerGrace+1, fails)
	assert.Equal(t, 0.0, lp.chargePower)

	// a reading that comes back is reported as a recovery, and only once
	assert.True(t, lp.chargePowerOK(), "coming back from the dark is announced")
	assert.False(t, lp.chargePowerOK(), "an ordinary reading is not")
}

// TestChargePowerGoesDarkFromZero: the transition matters, not the previous number. A resting device
// reads 0 W — and when its source disappears the topic must still stop claiming a reading (measured
// 2026-11-01: the first live run had a zero last value and produced no "going dark" line at all).
func TestChargePowerGoesDarkFromZero(t *testing.T) {
	lp := NewLoadpoint(util.NewLogger("foo"), settings.NewDatabaseSettingsAdapter("foo"))
	lp.chargeMeter = newChargeMeter(deadMeter{})
	lp.chargePower = 0

	for i := 1; i <= chargePowerGrace; i++ {
		fails, dark := lp.chargePowerFailed()
		assert.Equal(t, i, fails)
		assert.Equal(t, i == chargePowerGrace, dark, "exactly the grace tick is the transition")
	}
}
