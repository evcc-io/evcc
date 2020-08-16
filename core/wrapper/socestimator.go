package wrapper

import (
	"math"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

// SocEstimator provides vehicle soc and charge duration
// Vehicle SoC can be estimated to provide more granularity
type SocEstimator struct {
	log      *util.Logger
	vehicle  api.Vehicle
	estimate bool

	capacity          float64 // vehicle capacity in Wh cached to simplify testing
	socCharge         float64 // estimated vehicle SoC
	prevSoC           float64 // previous vehicle SoC in %
	prevChargedEnergy float64 // previous charged energy in Wh
	socPerWh          float64 // SoC percent per Wh
}

// NewSocEstimator creates new estimator
func NewSocEstimator(log *util.Logger, vehicle api.Vehicle, estimate bool) *SocEstimator {
	s := &SocEstimator{
		log:      log,
		vehicle:  vehicle,
		estimate: estimate,
	}

	s.Reset()

	return s
}

// Reset resets the estimation process to default values
func (s *SocEstimator) Reset() {
	s.prevSoC = -1                                   // indicate soc and energy invalid
	s.capacity = float64(s.vehicle.Capacity()) * 1e3 // cache to simplify debugging
	s.socPerWh = 100 / s.capacity
}

// RemainingChargeDuration returns the remaining duration estimate based on SoC, target and charge power
func (s *SocEstimator) RemainingChargeDuration(chargePower float64, targetSoC int) time.Duration {
	if chargePower > 0 {
		percentRemaining := float64(targetSoC) - s.socCharge
		if percentRemaining <= 0 {
			return 0
		}

		whRemaining := percentRemaining / 100 * s.capacity
		return time.Duration(float64(time.Hour) * whRemaining / chargePower).Round(time.Second)
	}

	return -1
}

// SoC implements Vehicle.ChargeState with addition of given charged energy
func (s *SocEstimator) SoC(chargedEnergy float64) (float64, error) {
	f, err := s.vehicle.ChargeState()
	if err != nil {
		return s.socCharge, err
	}

	s.socCharge = f

	if s.estimate {
		socDelta := s.socCharge - s.prevSoC
		energyDelta := chargedEnergy - s.prevChargedEnergy

		// soc or energy value changed (including unexpected energy reset)
		if socDelta != 0 || energyDelta != 0 {
			// starting point defined?
			if s.prevSoC > 0 {
				// re-calculate gradient, soc% per Wh
				if socDelta > 1 && energyDelta > 0 {
					s.socPerWh = socDelta / energyDelta
					s.log.TRACE.Printf("soc gradient: socPerWh %.3f%%/Wh, virtualBattery %.0f", s.socPerWh, s.socPerWh*s.capacity)
				} else {
					// soc unchanged, estimate
					s.socCharge = math.Min(f+energyDelta*s.socPerWh, 100)
					s.log.TRACE.Printf("soc estimated: %.2f%% (vehicle: %.0f%%)", s.socCharge, f)
				}
			}

			// sample charged energy at soc change
			if socDelta != 0 {
				s.prevChargedEnergy = chargedEnergy
				s.prevSoC = f
			}
		}
	}

	return s.socCharge, nil
}
