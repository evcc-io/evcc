package core

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

// TestLoadpointInflight verifies the un-metered reserve a loadpoint exposes to
// its circuit: max(0, intended - measured) during the settle window, then 0.
func TestLoadpointInflight(t *testing.T) {
	Voltage = 230

	clk := clock.NewMock()
	lp := &Loadpoint{
		log:    util.NewLogger("foo"),
		clock:  clk,
		phases: 1,
	}
	lp.enabled = true
	lp.offeredCurrent = 16                 // just raised the setpoint to 16A
	lp.chargeCurrents = []float64{6, 6, 6} // car still ramping, only drawing 6A
	lp.chargePower = currentToPower(6, 1)
	lp.actuatedAt = clk.Now()

	// within the settle window the un-metered part is reserved
	assert.Equal(t, 10.0, lp.GetInflightCurrent(), "current reserve = 16 - 6")
	assert.Equal(t, currentToPower(16, 1)-currentToPower(6, 1), lp.GetInflightPower())

	// after the settle window the meters are trusted again
	clk.Add(chargerSwitchDuration)
	assert.Zero(t, lp.GetInflightCurrent(), "reserve cleared after settle")
	assert.Zero(t, lp.GetInflightPower())
}

// TestLoadpointInflightDecrease verifies a reduction reserves nothing (the still
// higher metered draw is the conservative value the circuit already sees).
func TestLoadpointInflightDecrease(t *testing.T) {
	Voltage = 230

	clk := clock.NewMock()
	lp := &Loadpoint{
		log:    util.NewLogger("foo"),
		clock:  clk,
		phases: 1,
	}
	lp.enabled = true
	lp.offeredCurrent = 6                     // just lowered the setpoint to 6A
	lp.chargeCurrents = []float64{16, 16, 16} // still drawing 16A
	lp.chargePower = currentToPower(16, 1)
	lp.actuatedAt = clk.Now()

	assert.Zero(t, lp.GetInflightCurrent(), "decrease reserves nothing")
	assert.Zero(t, lp.GetInflightPower())
}
