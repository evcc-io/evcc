package soc

import (
	"errors"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const ChargeEfficiency = 0.9 // assume charge 90% efficiency

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
	s.prevSoc = 0
	s.prevChargedEnergy = 0
	s.initialSoc = 0
	s.capacity = float64(s.vehicle.Capacity()) * 1e3  // cache to simplify debugging
	s.virtualCapacity = s.capacity / ChargeEfficiency // initial capacity taking efficiency into account
	s.energyPerSocStep = s.virtualCapacity / 100
}

// RemainingChargeDuration returns the estimated remaining duration
func (s *Estimator) RemainingChargeDuration(targetSoc int, chargePower float64) time.Duration {
	energy := s.RemainingChargeEnergy(targetSoc) * 1e3 / chargePower
	if math.IsInf(energy, 0) {
		energy = 0
	}
	return time.Duration(float64(time.Hour) * energy).Round(time.Second)
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
func (s *Estimator) Soc(chargedEnergy float64) (float64, error) {
	var fetchedSoc *float64

	if charger, ok := s.charger.(api.Battery); ok {
		f, err := charger.Soc()

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
		f, err := s.vehicle.Soc()
		if err != nil {
			// required for online APIs with refreshkey
			if errors.Is(err, api.ErrMustRetry) {
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
			s.prevChargedEnergy = math.Max(chargedEnergy, 0)
			s.prevSoc = s.vehicleSoc
		} else {
			s.vehicleSoc = math.Min(*fetchedSoc+energyDelta/s.energyPerSocStep, 100)
			s.log.DEBUG.Printf("soc estimated: %.2f%% (vehicle: %.2f%%)", s.vehicleSoc, *fetchedSoc)
		}
	}

	return s.vehicleSoc, nil
}
