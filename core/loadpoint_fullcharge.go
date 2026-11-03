package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/vehicle"
)

// the battery management may still balance cells after the vehicle stopped charging at 100%
const fullChargeHold = 30 * time.Minute

// updateFullCharge raises the limit to 100% while a periodic full charge is due. A full charge
// is recorded once the vehicle charged and then stopped at 100% on its own for the hold time.
func (lp *Loadpoint) updateFullCharge() {
	v := lp.GetVehicle()
	energy := lp.GetChargedEnergy() / 1e3

	switch {
	case v != lp.fullChargeVehicle:
		// energy charged before the vehicle was identified is not attributed to it
		lp.fullChargeVehicle = v
		lp.fullChargeEnergy = energy
		lp.fullChargeCharged = false
		lp.fullChargeFull = time.Time{}
	case lp.GetStatus() == api.StatusA:
		lp.fullChargeCharged = false
		lp.fullChargeFull = time.Time{}
	case energy < lp.fullChargeEnergy:
		// a new session starts from zero
		lp.fullChargeEnergy = 0
	}

	delta := energy - lp.fullChargeEnergy
	lp.fullChargeEnergy = energy

	if v == nil || lp.chargerHasFeature(api.Heating) {
		lp.setFullChargeActive(false)
		return
	}

	vs := vehicle.Settings(lp.log, v)

	fc := vs.GetFullCharge()
	if !fc.Enabled() {
		lp.setFullChargeActive(false)
		return
	}

	if delta > 0 {
		vs.AddFullChargeEnergy(delta)
	}

	now := lp.clock.Now()

	last := vs.GetFullChargeTime()
	if last.IsZero() {
		last = now
		vs.SetFullChargeTime(now)
	}

	if lp.GetStatus() == api.StatusC {
		lp.fullChargeCharged = true
	}

	// charged, then stopped while still enabled: the vehicle decided it is full
	if lp.fullChargeCharged && lp.vehicleSoc >= 100 && lp.enabled && lp.GetStatus() == api.StatusB {
		if lp.fullChargeFull.IsZero() {
			lp.fullChargeFull = now
		}

		if now.Sub(lp.fullChargeFull) >= fullChargeHold {
			lp.log.INFO.Println("full charge: completed")
			vs.SetFullChargeTime(now)
			last = now
			lp.fullChargeCharged = false
			lp.fullChargeFull = time.Time{}
		}
	} else {
		lp.fullChargeFull = time.Time{}
	}

	due := fc.Days > 0 && !now.Before(last.AddDate(0, 0, fc.Days)) ||
		fc.Energy > 0 && vs.GetFullChargeEnergy() >= fc.Energy

	lp.setFullChargeActive(due)
}

func (lp *Loadpoint) setFullChargeActive(active bool) {
	lp.Lock()
	changed := lp.fullChargeActive != active
	lp.fullChargeActive = active
	lp.Unlock()

	lp.publish(keys.FullChargeActive, active)

	if changed {
		lp.log.INFO.Printf("full charge: due %t", active)
		lp.publish(keys.EffectiveLimitSoc, lp.EffectiveLimitSoc())
	}
}
