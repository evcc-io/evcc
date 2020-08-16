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

	socCharge         float64 // Vehicle SoC display (estimated)
	prevSoC           float64 // Previous vehicle SoC
	prevChargedEnergy float64 // Previous charged energy
	energyPerSocStep  float64 // Energy / SOC

}

func NewSocEstimator(log *util.Logger, vehicle api.Vehicle, estimate bool) *SocEstimator {
	s := &SocEstimator{
		log:      log,
		vehicle:  vehicle,
		estimate: estimate,
	}

	s.Reset()

	return s
}

func (s *SocEstimator) Reset() {
	s.prevSoC = -1
	s.prevChargedEnergy = 0
	s.energyPerSocStep = float64(s.vehicle.Capacity()) * 1e3 / 100
}

func (s *SocEstimator) RemainingChargeDuration(chargePower float64, targetSoC int) time.Duration {
	if chargePower > 0 {
		percentRemaining := float64(targetSoC) - s.socCharge
		if percentRemaining <= 0 {
			return 0
		}

		whTotal := float64(s.vehicle.Capacity()) * 1e3
		whRemaining := percentRemaining / 100 * whTotal
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
		energyDelta := chargedEnergy - s.prevChargedEnergy
		s.prevChargedEnergy = chargedEnergy

		s.log.TRACE.Printf("chargedEnergy: %.0fWh, energyDelta: %0.0fWh", chargedEnergy, energyDelta)

		// soc value updated
		if socDelta := f - s.prevSoC; socDelta > 0 {
			s.prevSoC = f

			// calculate gradient, wh per soc %
			if socDelta > 1 && energyDelta > 0 && s.prevSoC > 0 {
				s.energyPerSocStep = energyDelta / socDelta
				s.log.TRACE.Printf("soc gradient updated: energyPerSocStep: %0.0fWh, virtualBatCap: %0.1fkWh", s.energyPerSocStep, s.energyPerSocStep*100/1e3)
			}
		}

		s.socCharge = math.Min(s.prevSoC+(energyDelta/s.energyPerSocStep), 100)
		s.log.TRACE.Printf("soc estimated: %.2f%% (vehicle: %.0f%%)", s.socCharge, s.prevSoC)
	}

	return s.socCharge, nil
}
