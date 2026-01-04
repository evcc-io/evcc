package soc

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
)

// Estimator provides vehicle soc and charge duration
// Vehicle Soc can be estimated to provide more granularity
type Estimator struct {
	log      *util.Logger
	charger  api.Charger
	vehicle  api.Vehicle
	estimate bool

	capacity          float64 // vehicle capacity in Wh cached to simplify testing
	virtualCapacity   float64 // estimated virtual vehicle capacity in Wh
	vehicleSoc        float64 // estimated vehicle Soc
	initialSoc        float64 // first received valid vehicle Soc
	initialEnergy     float64 // energy counter at first valid Soc
	prevSoc           float64 // previous vehicle Soc in %
	prevChargedEnergy float64 // previous charged energy in Wh
	energyPerSocStep  float64 // Energy per Soc percent in Wh
	minChargePower    float64 // Lowest charge power (just before vehicle stops charging at 100%)
	maxChargePower    float64 // Highest charge power the battery can handle on any charger
	maxChargeSoc      float64 // SoC at/after which maxChargePower is degressive
}

// NewEstimator creates new estimator
func NewEstimator(log *util.Logger, charger api.Charger, vehicle api.Vehicle, estimate bool) *Estimator {
	s := &Estimator{
		log:      log,
		charger:  charger,
		vehicle:  vehicle,
		estimate: estimate,
	}

	s.Reset()

	return s
}

const (
	// power thresholds in watts
	householdOutletMaxPowerW     = 2300 // regular household outlet: 1-p / 10A / 230V
	singlePhaseWallboxMaxPowerW  = 3680 // wallbox: 1-p / 16A / 230V
)

// Computes the charge efficiency based on the charging power
func ChargeEfficiency(chargePower float64) float64 {
	if chargePower <= householdOutletMaxPowerW {
		return 0.85
	}
	if chargePower <= singlePhaseWallboxMaxPowerW {
		return 0.88
	}
	return 0.9 // wallbox: 1-p / 32A / 230V or 3-p / 16A / 400V or above
}

// Reset resets the estimation process to default values
func (s *Estimator) Reset() {
	s.prevSoc = 0
	s.prevChargedEnergy = 0
	s.initialSoc = 0
	s.capacity = s.vehicle.Capacity() * 1e3              // cache to simplify debugging
	s.virtualCapacity = s.capacity / ChargeEfficiency(0) // initial capacity taking efficiency into account
	s.energyPerSocStep = s.virtualCapacity / 100
	s.minChargePower = 1000  // default 1 kW
	s.maxChargePower = 50000 // default 50 kW
	s.maxChargeSoc = 50      // default 50%
}

// RemainingChargeDuration returns the estimated remaining duration
func (s *Estimator) RemainingChargeDuration(targetSoc int, chargePower float64) time.Duration {
	const minChargeSoc = 100

	//Update virtual capacity based on current chargePower
	s.virtualCapacity = s.capacity / ChargeEfficiency(chargePower)

	dy := s.minChargePower - s.maxChargePower
	dx := minChargeSoc - s.maxChargeSoc

	var rrp float64 = 100

	if dy < 0 && dx > 0 {
		m := dy / dx
		b := s.minChargePower - m*minChargeSoc

		// Relativer Reduktionspunkt
		rrp = (chargePower - b) / m
	}

	var t1, t2 float64

	// Zeit von vehicleSoc bis Reduktionspunkt (linear)
	if s.vehicleSoc <= rrp {
		t1 = (min(float64(targetSoc), rrp) - s.vehicleSoc) / minChargeSoc * s.virtualCapacity / chargePower
	}

	// Zeit von Reduktionspunkt bis targetSoc (degressiv)
	if float64(targetSoc) > rrp {
		t2 = (float64(targetSoc) - max(s.vehicleSoc, rrp)) / minChargeSoc * s.virtualCapacity / ((chargePower-s.minChargePower)/2 + s.minChargePower)
	}

	return max(0, time.Duration(float64(time.Hour)*(t1+t2))).Round(time.Second)
}

// RemainingChargeEnergy returns the remaining charge energy in kWh
func (s *Estimator) RemainingChargeEnergy(targetSoc int, chargePower float64) float64 {
	//Update virtual capacity based on current chargePower
	s.virtualCapacity = s.capacity / ChargeEfficiency(chargePower)

	percentRemaining := float64(targetSoc) - s.vehicleSoc
	if percentRemaining <= 0 || s.virtualCapacity <= 0 {
		return 0
	}

	// estimate remaining energy
	whRemaining := percentRemaining / 100 * s.virtualCapacity
	return whRemaining / 1e3
}

// Soc replaces the api.Vehicle.Soc interface to take charged energy into account
func (s *Estimator) Soc(chargedEnergy float64) (float64, error) {
	var fetchedSoc *float64

	if charger, ok := s.charger.(api.Battery); ok {
		f, err := Guard(charger.Soc())

		// if the charger does or could provide Soc, we always use it instead of using the vehicle API
		if err == nil || !errors.Is(err, api.ErrNotAvailable) {
			if err != nil {
				// never received a soc value
				if s.prevSoc == 0 {
					return 0, err
				}

				// recover from temporary api errors
				f = s.prevSoc
				s.log.WARN.Printf("vehicle soc (charger): %v (ignored by estimator)", err)
			}

			fetchedSoc = &f
			s.vehicleSoc = f
		}
	}

	if fetchedSoc == nil {
		f, err := Guard(s.vehicle.Soc())
		if err != nil {
			// required for online APIs with refreshkey
			if loadpoint.AcceptableError(err) {
				return 0, err
			}

			// never received a soc value
			if s.prevSoc == 0 {
				return 0, err
			}

			// recover from temporary api errors
			f = s.prevSoc
			s.log.WARN.Printf("vehicle soc: %v (ignored by estimator)", err)
		}

		fetchedSoc = &f
		s.vehicleSoc = f
	}

	if s.estimate && s.virtualCapacity > 0 {
		socDelta := s.vehicleSoc - s.prevSoc
		energyDelta := max(chargedEnergy, 0) - s.prevChargedEnergy

		if socDelta != 0 || energyDelta < 0 { // soc value change or unexpected energy reset
			// compare ChargeState of vehicle and charger
			var invalid bool

			if vs, ok := s.vehicle.(api.ChargeState); ok {
				ccs, err := s.charger.Status()
				if err != nil {
					return 0, err
				}
				vcs, err := vs.Status()
				if err != nil {
					vcs = ccs // sanitize vehicle errors
				} else {
					s.log.DEBUG.Printf("vehicle status: %s", vcs)
				}
				invalid = vcs != ccs
			}

			if !invalid {
				if s.initialSoc == 0 {
					s.initialSoc = s.vehicleSoc
					s.initialEnergy = chargedEnergy
				}

				socDiff := s.vehicleSoc - s.initialSoc
				energyDiff := chargedEnergy - s.initialEnergy

				// recalculate gradient, wh per soc %
				if socDiff > 10 && energyDiff > 0 {
					s.energyPerSocStep = energyDiff / socDiff
					s.virtualCapacity = s.energyPerSocStep * 100
					s.log.DEBUG.Printf("soc gradient updated: soc: %.1f%%, socDiff: %.1f%%, energyDiff: %.0fWh, energyPerSocStep: %.1fWh, virtualCapacity: %.0fWh", s.vehicleSoc, socDiff, energyDiff, s.energyPerSocStep, s.virtualCapacity)
				}
			}

			// sample charged energy at soc change, reset energy delta
			s.prevChargedEnergy = max(chargedEnergy, 0)
			s.prevSoc = s.vehicleSoc
		} else {
			s.vehicleSoc = min(*fetchedSoc+energyDelta/s.energyPerSocStep, 100)
			s.log.DEBUG.Printf("soc estimated: %.2f%% (vehicle: %.2f%%)", s.vehicleSoc, *fetchedSoc)
		}
	}

	return s.vehicleSoc, nil
}
