package core

import (
	"fmt"
	"time"
)

const (
	pvTimer   = "pv"
	pvEnable  = "enable"
	pvDisable = "disable"

	guardTimer  = "guard"
	guardEnable = "enable"

	phaseTimer   = "phase"
	phaseScale1p = "scale1p"
	phaseScale3p = "scale3p"

	timerInactive = "inactive"
)

// elapseGuardTimer puts the guard timer into elapsed state
func (lp *Loadpoint) elapseGuardTimer() {
	if lp.guardUpdated.Equal(elapsed) {
		return
	}

	lp.log.DEBUG.Print("guard timer elapse")

	lp.guardUpdated = elapsed
	lp.publishTimer(guardTimer, 0, timerInactive)
}

// elapsePVTimer puts the pv enable/disable timer into elapsed state
func (lp *Loadpoint) elapsePVTimer() {
	if lp.pvTimer.Equal(elapsed) {
		return
	}

	lp.log.DEBUG.Printf("pv timer elapse")

	lp.pvTimer = elapsed
	lp.publishTimer(pvTimer, 0, timerInactive)

	lp.elapseGuardTimer()
}

// resetPVTimer resets the pv enable/disable timer to inactive state
func (lp *Loadpoint) resetPVTimer(typ ...string) {
	if lp.pvTimer.IsZero() {
		return
	}

	msg := "pv timer reset"
	if len(typ) == 1 {
		msg = fmt.Sprintf("pv %s timer reset", typ[0])
	}
	lp.log.DEBUG.Printf(msg)

	lp.pvTimer = time.Time{}
	lp.publishTimer(pvTimer, 0, timerInactive)
}

// resetPhaseTimer resets the phase switch timer to inactive state
func (lp *Loadpoint) resetPhaseTimer() {
	if lp.phaseTimer.IsZero() {
		return
	}

	lp.log.DEBUG.Printf("phase timer reset")

	lp.phaseTimer = time.Time{}
	lp.publishTimer(phaseTimer, 0, timerInactive)
}

// publishTimer sends timer updates to the ui
func (lp *Loadpoint) publishTimer(name string, delay time.Duration, action string) {
	timer := lp.pvTimer
	if name == phaseTimer {
		timer = lp.phaseTimer
	}
	if name == guardTimer {
		timer = lp.guardUpdated
	}

	remaining := delay - lp.clock.Since(timer)
	if remaining < 0 {
		remaining = 0
	}

	lp.publish(name+"Action", action)
	lp.publish(name+"Remaining", remaining)

	if action == timerInactive {
		lp.log.DEBUG.Printf("%s timer %s", name, action)
	} else {
		lp.log.DEBUG.Printf("%s %s in %v", name, action, remaining.Round(time.Second))
	}
}
