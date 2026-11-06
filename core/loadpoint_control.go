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

// requestControl requests site to invoke loadpoint.Control() without waiting for the next interval
func (lp *Loadpoint) requestControl() {
	lp.pendingControl.Store(true)

	// a full slot already signals a pending request
	select {
	case lp.controlChan <- struct{}{}:
	default:
	}
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
