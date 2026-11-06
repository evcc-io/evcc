package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/grid"
	"github.com/stretchr/testify/assert"
)

// TestGetInflightPower verifies the self-correcting surplus reserve a loadpoint
// exposes to the site: max(0, intended - measured) during the settle window,
// collapsing to zero as soon as the meter catches up (or the window expires).
func TestGetInflightPower(t *testing.T) {
	clk := clock.NewMock()
	lp := &Loadpoint{log: util.NewLogger("foo"), clock: clk, phases: 1, controlInterval: time.Minute}
	lp.enabled = true
	lp.offeredCurrent = 16                     // just raised the setpoint to 16A
	lp.chargePower = grid.CurrentToPower(6, 1) // car still ramping, only drawing 6A
	lp.actuatedAt = clk.Now()

	assert.Equal(t, grid.CurrentToPower(16, 1)-grid.CurrentToPower(6, 1), lp.GetInflightPower(), "reserve = intended - measured")

	// self-correcting: once the meter catches up, the reserve collapses even
	// within the window (the property the delta-ledger lacked)
	lp.chargePower = grid.CurrentToPower(16, 1)
	assert.Zero(t, lp.GetInflightPower(), "reserve collapses when the meter catches up")

	// and after the window the meters are trusted again regardless
	lp.chargePower = grid.CurrentToPower(6, 1)
	clk.Add(lp.controlInterval)
	assert.Zero(t, lp.GetInflightPower(), "reserve cleared after the settle window")
}

// TestGetInflightPowerDecrease verifies a reduction reserves nothing (the still
// higher metered draw is the conservative value the site already sees).
func TestGetInflightPowerDecrease(t *testing.T) {
	clk := clock.NewMock()
	lp := &Loadpoint{log: util.NewLogger("foo"), clock: clk, phases: 1, controlInterval: time.Minute}
	lp.enabled = true
	lp.offeredCurrent = 6                       // just lowered the setpoint to 6A
	lp.chargePower = grid.CurrentToPower(16, 1) // still drawing 16A
	lp.actuatedAt = clk.Now()

	assert.Zero(t, lp.GetInflightPower(), "decrease reserves nothing")
}

// TestForeignInflightPower verifies the site discounts the surplus only by the
// reserve of other loadpoints, never by the controlled loadpoint's own reserve.
func TestForeignInflightPower(t *testing.T) {
	clk := clock.NewMock()
	newLp := func(current float64) *Loadpoint {
		lp := &Loadpoint{log: util.NewLogger("foo"), clock: clk, phases: 1, controlInterval: time.Minute}
		lp.enabled = true
		lp.offeredCurrent = current
		lp.actuatedAt = clk.Now()
		return lp
	}

	lp1, lp2 := newLp(16), newLp(10)
	site := &Site{loadpoints: []*Loadpoint{lp1, nil, lp2}}

	assert.Equal(t, lp2.GetInflightPower(), site.foreignInflightPower(lp1))
	assert.Equal(t, lp1.GetInflightPower(), site.foreignInflightPower(lp2))
}

// TestGetInflightPowerAfterPhaseSwitch verifies the reserve is sized with the
// post-switch phases: the switch resets the measured phases, so the stale 1p
// measurement does not shrink the 3p reserve during the phase switch window.
func TestGetInflightPowerAfterPhaseSwitch(t *testing.T) {
	clk := clock.NewMock()
	lp := &Loadpoint{log: util.NewLogger("foo"), clock: clk, phases: 1, controlInterval: time.Minute}
	lp.enabled = true
	lp.offeredCurrent = 6
	lp.measuredPhases = 1
	lp.chargePower = 0 // charging pauses during the switch
	lp.actuatedAt = clk.Now()

	assert.Equal(t, grid.CurrentToPower(6, 1), lp.GetInflightPower(), "1p before the switch")

	lp.phasesSwitched = clk.Now()
	lp.SetPhases(3)

	assert.Equal(t, grid.CurrentToPower(6, 3), lp.GetInflightPower(), "3p right after the switch")
}
