package soc

import (
	"errors"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const chargeEfficiency = 0.9 // assume charge 90% efficiency

// Estimator provides vehicle soc and charge duration
// Vehicle SoC can be estimated to provide more granularity
type Estimator struct {
	log      *util.Logger
	charger  api.Charger
	vehicle  api.Vehicle
	estimate bool

	capacity          float64 // vehicle capacity in Wh cached to simplify testing
	virtualCapacity   float64 // estimated virtual vehicle capacity in Wh
	vehicleSoc        float64 // estimated vehicle SoC
	prevSoC           float64 // previous vehicle SoC in %
	prevChargedEnergy float64 // previous charged energy in Wh
	energyPerSocStep  float64 // Energy per SoC percent in Wh
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

// Reset resets the estimation process to default values
func (s *Estimator) Reset() {
	s.prevSoC = 0
	s.prevChargedEnergy = 0
	s.capacity = float64(s.vehicle.Capacity()) * 1e3  // cache to simplify debugging
	s.virtualCapacity = s.capacity / chargeEfficiency // initial capacity taking efficiency into account
	s.energyPerSocStep = s.virtualCapacity / 100
}

// AssumedChargeDuration estimates charge duration up to targetSoC based on virtual capacity
func (s *Estimator) AssumedChargeDuration(chargePower float64, targetSoC int) time.Duration {
	percentRemaining := float64(targetSoC) - s.vehicleSoc

	if percentRemaining <= 0 {
		return 0
	}

	whRemaining := percentRemaining / 100 * s.virtualCapacity
	return time.Duration(float64(time.Hour) * whRemaining / chargePower).Round(time.Second)
}

// RemainingChargeDuration returns the remaining duration estimate based on SoC, target and charge power
func (s *Estimator) RemainingChargeDuration(chargePower float64, targetSoC int) time.Duration {
	if chargePower > 0 {
		percentRemaining := float64(targetSoC) - s.vehicleSoc
		if percentRemaining <= 0 {
			return 0
		}

		// use vehicle api if available
		if vr, ok := s.vehicle.(api.VehicleFinishTimer); ok {
			finishTime, err := vr.FinishTime()
			if err == nil {
				timeRemaining := time.Until(finishTime)
				return time.Duration(float64(timeRemaining) * percentRemaining / (100 - s.vehicleSoc))
			}

			if !errors.Is(err, api.ErrNotAvailable) {
				s.log.WARN.Printf("updating remaining time failed: %v", err)
			}
		}

		return s.AssumedChargeDuration(chargePower, targetSoC)
	}

	return -1
}

// RemainingChargeEnergy returns the remaining charge energy in kWh
func (s *Estimator) RemainingChargeEnergy(targetSoC int) float64 {
	percentRemaining := float64(targetSoC) - s.vehicleSoc
	if percentRemaining <= 0 {
		return 0
	}

	// estimate remaining energy
	whRemaining := percentRemaining / 100 * s.virtualCapacity
	return whRemaining / 1e3
}

// SoC replaces the api.Vehicle.SoC interface to take charged energy into account
func (s *Estimator) SoC(chargedEnergy float64) (float64, error) {
	var fetchedSoC *float64

	if charger, ok := s.charger.(api.Battery); ok {
		f, err := charger.SoC()

		// if the charger does or could provide SoC, we always use it instead of using the vehicle API
		if err == nil || !errors.Is(err, api.ErrNotAvailable) {
			if err != nil {
				// never received a soc value
				if s.prevSoC == 0 {
					return 0, err
				}

				// recover from temporary api errors
				f = s.prevSoC
				s.log.WARN.Printf("vehicle soc (charger): %v (ignored by estimator)", err)
			}

			fetchedSoC = &f
			s.vehicleSoc = f
		}
	}

	if fetchedSoC == nil {
		f, err := s.vehicle.SoC()
		if err != nil {
			// required for online APIs with refreshkey
			if errors.Is(err, api.ErrMustRetry) {
				return 0, err
			}

			// never received a soc value
			if s.prevSoC == 0 {
				return 0, err
			}

			// recover from temporary api errors
			f = s.prevSoC
			s.log.WARN.Printf("vehicle soc: %v (ignored by estimator)", err)
		}

		fetchedSoC = &f
		s.vehicleSoc = f
	}

	if s.estimate {
		socDelta := s.vehicleSoc - s.prevSoC
		energyDelta := math.Max(chargedEnergy, 0) - s.prevChargedEnergy

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

			// calculate gradient, wh per soc %
			if !invalid && socDelta > 2 && energyDelta > 0 && s.prevSoC > 0 {
				s.energyPerSocStep = energyDelta / socDelta
				s.virtualCapacity = s.energyPerSocStep * 100
				s.log.DEBUG.Printf("soc gradient updated: energyPerSocStep: %0.0fWh, virtualCapacity: %0.0fWh", s.energyPerSocStep, s.virtualCapacity)
			}

			// sample charged energy at soc change, reset energy delta
			s.prevChargedEnergy = math.Max(chargedEnergy, 0)
			s.prevSoC = s.vehicleSoc
		} else {
			s.vehicleSoc = math.Min(*fetchedSoC+energyDelta/s.energyPerSocStep, 100)
			s.log.DEBUG.Printf("soc estimated: %.2f%% (vehicle: %.2f%%)", s.vehicleSoc, *fetchedSoC)
		}
	}

	return s.vehicleSoc, nil
}
