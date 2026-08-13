package soc

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	ChargeEfficiency = 0.85 // assume 85% charge efficiency

	minChargePower = 1000.0  // charge power at 100% soc (just before the vehicle stops charging)
	maxChargePower = 50000.0 // charge power up to maxChargeSoc
	maxChargeSoc   = 50.0    // soc up to which maxChargePower is available

	// power reduction per soc percent above maxChargeSoc
	powerPerSoc = (maxChargePower - minChargePower) / (100 - maxChargeSoc)
)

// Estimator provides vehicle soc and charge duration
// Vehicle Soc can be estimated to provide more granularity
type Estimator struct {
	log *util.Logger

	capacity          float64 // vehicle capacity in Wh
	energyPerSocStep  float64 // energy per soc percent in Wh
	vehicleSoc        float64 // estimated vehicle soc in %
	initialSoc        float64 // first received valid vehicle soc in %
	initialEnergy     float64 // energy counter at first valid soc in Wh
	prevSoc           float64 // vehicle soc at last soc change in %
	prevChargedEnergy float64 // charged energy at last soc change in Wh
}

// NewEstimator creates new estimator
func NewEstimator(log *util.Logger, vehicle api.Vehicle) *Estimator {
	capacity := vehicle.Capacity() * 1e3

	return &Estimator{
		log:              log,
		capacity:         capacity,
		energyPerSocStep: capacity / ChargeEfficiency / 100, // initial gradient taking efficiency into account
	}
}

// virtualCapacity returns the estimated capacity in Wh, never below the vehicle's physical capacity
func (s *Estimator) virtualCapacity() float64 {
	return max(s.capacity, s.energyPerSocStep*100)
}

// RemainingChargeDuration returns the estimated remaining duration
func (s *Estimator) RemainingChargeDuration(targetSoc, chargePower float64) time.Duration {
	return remainingChargeDuration(targetSoc, chargePower, s.vehicleSoc, s.virtualCapacity())
}

func RemainingChargeDuration(targetSoc, chargePower, vehicleSoc, capacity float64) time.Duration {
	return remainingChargeDuration(targetSoc, chargePower, vehicleSoc, capacity*1e3/ChargeEfficiency)
}

func remainingChargeDuration(targetSoc, chargePower, vehicleSoc, virtualCapacity float64) time.Duration {
	// soc above which charge power starts to taper off
	taperSoc := 100 - (chargePower-minChargePower)/powerPerSoc

	var hours float64

	// below the taper point the vehicle charges at full power
	if vehicleSoc < taperSoc {
		hours += (min(targetSoc, taperSoc) - vehicleSoc) / 100 * virtualCapacity / chargePower
	}

	// above the taper point power decreases linearly towards minChargePower
	if targetSoc > taperSoc {
		hours += (targetSoc - max(vehicleSoc, taperSoc)) / 100 * virtualCapacity / ((chargePower + minChargePower) / 2)
	}

	return max(0, time.Duration(float64(time.Hour)*hours)).Round(time.Second)
}

// RemainingChargeEnergy returns the remaining charge energy in kWh
func (s *Estimator) RemainingChargeEnergy(targetSoc int) float64 {
	return remainingChargeEnergy(float64(targetSoc), s.vehicleSoc, s.virtualCapacity())
}

func RemainingChargeEnergy(targetSoc int, vehicleSoc, capacity float64) float64 {
	return remainingChargeEnergy(float64(targetSoc), vehicleSoc, capacity*1e3/ChargeEfficiency)
}

func remainingChargeEnergy(targetSoc, vehicleSoc, virtualCapacity float64) float64 {
	return max(0, targetSoc-vehicleSoc) / 100 * max(0, virtualCapacity) / 1e3
}

// Soc replaces the api.Vehicle.Soc interface to take charged energy into account
func (s *Estimator) Soc(fetchedSoc *float64, chargedEnergy float64) float64 {
	if fetchedSoc == nil {
		s.log.WARN.Println("missing vehicle soc- ignored by estimator")
		return s.vehicleSoc
	}

	chargedEnergy = max(chargedEnergy, 0)
	socDelta := *fetchedSoc - s.prevSoc
	energyDelta := chargedEnergy - s.prevChargedEnergy

	// no soc change and no energy reset: interpolate soc from charged energy
	if socDelta == 0 && energyDelta >= 0 {
		s.vehicleSoc = min(*fetchedSoc+energyDelta/s.energyPerSocStep, 100)
		s.log.DEBUG.Printf("soc estimated: %.2f%% (vehicle: %.2f%%)", s.vehicleSoc, *fetchedSoc)
		return s.vehicleSoc
	}

	s.vehicleSoc = *fetchedSoc

	if s.initialSoc == 0 {
		s.initialSoc = s.vehicleSoc
		s.initialEnergy = chargedEnergy
	}

	socDiff := s.vehicleSoc - s.initialSoc
	energyDiff := chargedEnergy - s.initialEnergy

	// recalculate gradient, wh per soc %
	if socDiff > 10 && energyDiff > 0 {
		s.energyPerSocStep = energyDiff / socDiff
		s.log.DEBUG.Printf("soc gradient updated: soc: %.1f%%, socDiff: %.1f%%, energyDiff: %.0fWh, energyPerSocStep: %.1fWh, virtualCapacity: %.0fWh", s.vehicleSoc, socDiff, energyDiff, s.energyPerSocStep, s.virtualCapacity())
	}

	// sample charged energy at soc change
	s.prevSoc = s.vehicleSoc
	s.prevChargedEnergy = chargedEnergy

	return s.vehicleSoc
}
