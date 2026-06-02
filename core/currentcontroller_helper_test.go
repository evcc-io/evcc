package core

// currentController wraps a loadpoint in a CurrentController for current-domain unit tests
func currentController(lp *Loadpoint) *CurrentController {
	return &CurrentController{Loadpoint: lp}
}
