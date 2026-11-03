package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/vehicle"
)

const (
	// the battery management may still balance cells after the vehicle stopped charging at 100%
	fullChargeHold = 30 * time.Minute

	fullChargeEnergyInterval = 5 * time.Minute
)

// fullChargeEnergyMemo caches the energy of the vehicle's sessions since the last full charge
type fullChargeEnergyMemo struct {
	since   time.Time
	updated time.Time
	energy  float64 // kWh
}

// updateFullCharge raises the limit to 100% while a periodic full charge is due. A full charge
// is recorded once the vehicle stopped charging at 100% on its own for the hold time.
func (lp *Loadpoint) updateFullCharge() {
	v := lp.GetVehicle()
	if v == nil || lp.chargerHasFeature(api.Heating) {
		lp.fullChargeFull = time.Time{}
		lp.setFullChargeActive(false)
		return
	}

	vs := vehicle.Settings(lp.log, v)

	fc := vs.GetFullCharge()
	if !fc.Enabled() {
		lp.fullChargeFull = time.Time{}
		lp.setFullChargeActive(false)
		return
	}

	now := lp.clock.Now()

	last := vs.GetFullChargeTime()
	if last.IsZero() {
		last = now
		vs.SetFullChargeTime(now)
	}

	// charging stopped while still enabled, the vehicle decided it is full
	if lp.vehicleSoc >= 100 && lp.enabled && lp.GetStatus() == api.StatusB {
		if lp.fullChargeFull.IsZero() {
			lp.fullChargeFull = now
		}

		// record once per stop, a vehicle remaining at 100% does not move the date
		if now.Sub(lp.fullChargeFull) >= fullChargeHold && last.Before(lp.fullChargeFull) {
			lp.log.INFO.Println("full charge: completed")
			vs.SetFullChargeTime(now)
			last = now
		}
	} else {
		lp.fullChargeFull = time.Time{}
	}

	due := fc.Days > 0 && !now.Before(last.AddDate(0, 0, fc.Days)) ||
		fc.Energy > 0 && lp.fullChargeEnergySince(v.GetTitle(), last) >= fc.Energy

	lp.setFullChargeActive(due)
}

// fullChargeEnergySince returns the energy in kWh the vehicle charged since the given time
func (lp *Loadpoint) fullChargeEnergySince(title string, since time.Time) float64 {
	m := &lp.fullChargeEnergy
	now := lp.clock.Now()

	if lp.db != nil && (!m.since.Equal(since) || now.Sub(m.updated) >= fullChargeEnergyInterval) {
		sessions, err := lp.db.Sessions()
		if err != nil {
			lp.log.ERROR.Printf("full charge: %v", err)
		} else {
			var energy float64
			for _, s := range sessions {
				if s.Vehicle == title && !s.Created.Before(since) && (lp.session == nil || s.ID != lp.session.ID) {
					energy += s.ChargedEnergy
				}
			}

			*m = fullChargeEnergyMemo{since: since, updated: now, energy: energy}
		}
	}

	res := m.energy

	// the running session stores its energy when it stops
	if s := lp.session; s != nil && !s.Created.IsZero() && !s.Created.Before(since) {
		res += lp.GetChargedEnergy() / 1e3
	}

	return res
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
