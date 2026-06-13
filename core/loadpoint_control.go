package core

import "cmp"

// controlPending returns control request status to site
func (lp *Loadpoint) controlPending() bool {
	return lp.pendingControl.Load()
}

// controlDone clears pending control request
func (lp *Loadpoint) controlDone() {
	lp.pendingControl.Store(false)
}

// requestControl requests site to invoke loadpoint.Control()
// TODO check if all call sites are required
func (lp *Loadpoint) requestControl() {
	lp.pendingControl.Store(true)
}

// inflightActive reports whether a just-actuated setpoint is still settling, so
// the meters do not yet reflect it. The caller must hold the read lock.
func (lp *Loadpoint) inflightActive() bool {
	// settleDuration is the control interval; fall back to chargerSwitchDuration
	// if interval was never set (loadpoint not run via Site.Run) so the reserve
	// is never silently disabled.
	// TODO how can this happen
	settleDuration := cmp.Or(lp.controlInterval, chargerSwitchDuration)
	return !lp.actuatedAt.IsZero() && lp.clock.Since(lp.actuatedAt) < settleDuration
}
