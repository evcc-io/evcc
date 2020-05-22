package core

import (
	"time"

	"github.com/andig/evcc/core/wrapper"
)

// nilVal implements Stringer for nil values
type nilVal int

func (n *nilVal) String() string {
	return "—"
}

func (lp *LoadPoint) hasChargeMeter() bool {
	_, isWrapped := lp.chargeMeter.(*wrapper.ChargeMeter)
	return lp.chargeMeter != nil && !isWrapped
}

// chargeDuration returns for how long the charge cycle has been running
func (lp *LoadPoint) chargeDuration() time.Duration {
	d, err := lp.chargeTimer.ChargingTime()
	if err != nil {
		log.ERROR.Printf("%s charge timer error: %v", lp.Name, err)
	}
	return d
}

// chargedEnergy returns energy consumption since charge start in kWh
func (lp *LoadPoint) chargedEnergy() float64 {
	f, err := lp.chargeRater.ChargedEnergy()
	if err != nil {
		log.ERROR.Printf("%s charge rater error: %v", lp.Name, err)
	}
	return f
}

// remainingChargeDuration returns the remaining charge time
func (lp *LoadPoint) remainingChargeDuration(chargePercent float64) time.Duration {
	if !lp.charging {
		return -1
	}

	if lp.chargePower > 0 && lp.vehicle != nil {
		whRemaining := (1 - chargePercent/100.0) * 1e3 * float64(lp.vehicle.Capacity())
		return time.Duration(float64(time.Hour) * whRemaining / lp.chargePower)
	}

	return -1
}

// publish state of charge and remaining charge duration
func (lp *LoadPoint) publishSoC() {
	if lp.vehicle == nil {
		return
	}

	if lp.connected() {
		f, err := lp.vehicle.ChargeState()
		if err == nil {
			log.DEBUG.Printf("%s vehicle soc: %.1f%%", lp.Name, f)
			lp.publish("socCharge", f)
			lp.publish("chargeEstimate", lp.remainingChargeDuration(f))
			return
		}
		log.ERROR.Printf("%s vehicle error: %v", lp.Name, err)
	}

	var n *nilVal
	lp.publish("socCharge", n)
	lp.publish("chargeEstimate", -1)
}
