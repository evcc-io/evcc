package core

// currentController wraps a loadpoint in a CurrentController for current-domain unit tests
func currentController(lp *Loadpoint) *CurrentController {
	if ctrl, ok := lp.chargeController.(*CurrentController); ok {
		return ctrl
	}
	ctrl := newCurrentController(lp)
	lp.chargeController = ctrl
	return ctrl
}
