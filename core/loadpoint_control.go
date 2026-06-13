package core

func (lp *Loadpoint) controlPending() bool {
	return lp.pendingControl.Load()
}

func (lp *Loadpoint) controlDone() {
	lp.pendingControl.Store(false)
}

// inflightActive reports whether a just-actuated setpoint is still settling, so
// the meters do not yet reflect it. The caller must hold the read lock.
func (lp *Loadpoint) inflightActive() bool {
	// settle window is the control interval; fall back to chargerSwitchDuration
	// if interval was never set (loadpoint not run via Site.Run) so the reserve
	// is never silently disabled.
	window := lp.controlInterval
	if window <= 0 {
		window = chargerSwitchDuration
	}
	return !lp.actuatedAt.IsZero() && lp.clock.Since(lp.actuatedAt) < window
}
