package soc

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	ChargeEfficiency = 0.85 // assume 85% charge efficiency

	minChargePower = 1000.0  // Lowest charge power (just before vehicle stops charging at 100%)
	maxChargePower = 50000.0 // default 50 kW
	maxChargeSoc   = 50.0    // default 50%
	minChargeSoc   = 100.0

	gradient = (minChargePower - maxChargePower) / (minChargeSoc - maxChargeSoc)
)

// Estimator provides vehicle soc and charge duration
// Vehicle Soc can be estimated to provide more granularity
type Estimator struct {
	log     *util.Logger
	charger api.Charger
	vehicle api.Vehicle

	capacity          float64 // vehicle capacity in Wh cached to simplify testing
	virtualCapacity   float64 // estimated virtual vehicle capacity in Wh
	vehicleSoc        float64 // estimated vehicle Soc
	initialSoc        float64 // first received valid vehicle Soc
	initialEnergy     float64 // energy counter at first valid Soc
	prevSoc           float64 // previous vehicle Soc in %
	prevChargedEnergy float64 // previous charged energy in Wh
	energyPerSocStep  float64 // Energy per Soc percent in Wh
}

// NewEstimator creates new estimator
func NewEstimator(log *util.Logger, charger api.Charger, vehicle api.Vehicle) *Estimator {
	s := &Estimator{
		log:     log,
		charger: charger,
		vehicle: vehicle,
	}

	s.Reset()

	return s
}

// Reset resets the estimation process to default values
func (s *Estimator) Reset() {
	s.prevSoc = 0
	s.prevChargedEnergy = 0
	s.initialSoc = 0
	s.capacity = s.vehicle.Capacity() * 1e3           // cache to simplify debugging
	s.virtualCapacity = s.capacity / ChargeEfficiency // initial capacity taking efficiency into account
	s.energyPerSocStep = s.virtualCapacity / 100
}

// RemainingChargeDuration returns the estimated remaining duration
func (s *Estimator) RemainingChargeDuration(targetSoc, chargePower float64) time.Duration {
	return remainingChargeDuration(targetSoc, chargePower, s.vehicleSoc, s.virtualCapacity)
}

func RemainingChargeDuration(targetSoc, chargePower, vehicleSoc, virtualCapacity float64) time.Duration {
	return remainingChargeDuration(targetSoc, chargePower, vehicleSoc, virtualCapacity*1e3/ChargeEfficiency)
}

func remainingChargeDuration(targetSoc, chargePower, vehicleSoc, virtualCapacity float64) time.Duration {
	// Relativer Reduktionspunkt
	rrp := (chargePower-minChargePower)/gradient + minChargeSoc

	var t1, t2 float64

	// Zeit von vehicleSoc bis Reduktionspunkt (linear)
	if vehicleSoc < rrp {
		t1 = (min(float64(targetSoc), rrp) - vehicleSoc) / minChargeSoc * virtualCapacity / chargePower
	}

	// Zeit von Reduktionspunkt bis targetSoc (degressiv)
	if float64(targetSoc) > rrp {
		t2 = (float64(targetSoc) - max(vehicleSoc, rrp)) / minChargeSoc * virtualCapacity / ((chargePower-minChargePower)/2 + minChargePower)
	}

	return max(0, time.Duration(float64(time.Hour)*(t1+t2))).Round(time.Second)
}

// RemainingChargeEnergy returns the remaining charge energy in kWh
func (s *Estimator) RemainingChargeEnergy(targetSoc int) float64 {
	percentRemaining := float64(targetSoc) - s.vehicleSoc
	if percentRemaining <= 0 || s.virtualCapacity <= 0 {
		return 0
	}

	// estimate remaining energy
	whRemaining := percentRemaining / 100 * s.virtualCapacity
	return whRemaining / 1e3
}

// Soc replaces the api.Vehicle.Soc interface to take charged energy into account
func (s *Estimator) Soc(fetchedSoc *float64, chargedEnergy float64) (float64, error) {
	if fetchedSoc != nil {
		s.vehicleSoc = *fetchedSoc
	} else {
		s.log.WARN.Printf("missing vehicle soc- ignored by estimator")
	}

	if s.virtualCapacity > 0 {
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
