package core

import (
	"github.com/evcc-io/evcc/api"
)

// currentControllerAdapter adapts the loadpoint's existing current-based control
// to the chargercontroller.Controller interface. It converts power targets to
// current and delegates to the loadpoint's existing setLimit/fastCharging methods.
//
// This is a transitional adapter. The long-term goal is to extract the current/phase
// logic into a standalone CurrentController in core/chargercontroller/.
type currentControllerAdapter struct {
	lp *Loadpoint
}

func (a *currentControllerAdapter) SetOfferedPower(power float64) error {
	activePhases := a.lp.ActivePhases()
	current := powerToCurrent(power, activePhases)
	return a.lp.setLimit(current)
}

func (a *currentControllerAdapter) SetMaxPower() error {
	return a.lp.fastCharging()
}

func (a *currentControllerAdapter) MinPower() float64 {
	return Voltage * a.lp.effectiveMinCurrent() * float64(a.lp.minActivePhases())
}

func (a *currentControllerAdapter) MaxPower() float64 {
	return a.lp.effectiveMaxPower()
}

func (a *currentControllerAdapter) EffectiveChargePower() float64 {
	if a.lp.chargerHasFeature(api.IntegratedDevice) {
		return a.lp.chargePower
	}
	activePhases := a.lp.ActivePhases()
	return a.lp.effectiveCurrent() * float64(activePhases) * Voltage
}
