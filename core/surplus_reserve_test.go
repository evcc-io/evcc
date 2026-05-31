package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

// TestGetInflightPower verifies the self-correcting surplus reserve a loadpoint
// exposes to the site: max(0, intended - measured) during the settle window,
// collapsing to zero as soon as the meter catches up (or the window expires).
func TestGetInflightPower(t *testing.T) {
	Voltage = 230

	clk := clock.NewMock()
	lp := &Loadpoint{log: util.NewLogger("foo"), clock: clk, phases: 1, interval: time.Minute}
	lp.enabled = true
	lp.offeredCurrent = 16                // just raised the setpoint to 16A
	lp.chargePower = currentToPower(6, 1) // car still ramping, only drawing 6A
	lp.actuatedAt = clk.Now()

	assert.Equal(t, currentToPower(16, 1)-currentToPower(6, 1), lp.GetInflightPower(), "reserve = intended - measured")

	// self-correcting: once the meter catches up, the reserve collapses even
	// within the window (the property the delta-ledger lacked)
	lp.chargePower = currentToPower(16, 1)
	assert.Zero(t, lp.GetInflightPower(), "reserve collapses when the meter catches up")

	// and after the window the meters are trusted again regardless
	lp.chargePower = currentToPower(6, 1)
	clk.Add(lp.interval)
	assert.Zero(t, lp.GetInflightPower(), "reserve cleared after the settle window")
}

// TestGetInflightPowerDecrease verifies a reduction reserves nothing (the still
// higher metered draw is the conservative value the site already sees).
func TestGetInflightPowerDecrease(t *testing.T) {
	Voltage = 230

	clk := clock.NewMock()
	lp := &Loadpoint{log: util.NewLogger("foo"), clock: clk, phases: 1, interval: time.Minute}
	lp.enabled = true
	lp.offeredCurrent = 6                  // just lowered the setpoint to 6A
	lp.chargePower = currentToPower(16, 1) // still drawing 16A
	lp.actuatedAt = clk.Now()

	assert.Zero(t, lp.GetInflightPower(), "decrease reserves nothing")
}
