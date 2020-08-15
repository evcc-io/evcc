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

func (s *SocEstimator) RemainingChargeDuration(chargePercent float64) time.Duration {
	if !s.charging {
		return -1
	}

	if s.chargePower > 0 && s.vehicle != nil {
		chargePercent = chargePercent / 100.0
		targetPercent := float64(s.TargetSoC) / 100

		if chargePercent >= targetPercent {
			return 0
		}

		whTotal := float64(s.vehicle.Capacity()) * 1e3
		whRemaining := (targetPercent - chargePercent) * whTotal
		return time.Duration(float64(time.Hour) * whRemaining / s.chargePower).Round(time.Second)
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
		socDelta := f - s.prevSoC
		s.prevSoC = f

		energyDelta := chargedEnergy - s.prevChargedEnergy

		// soc value updated
		if socDelta > 0 || energyDelta < 0 {
			// calculate gradient, wh per soc %
			if (s.prevSoC > 0) && (socDelta >= 2) && (energyDelta > 0) {
				s.energyPerSocStep = energyDelta / socDelta
			}

			s.prevChargedEnergy = chargedEnergy
			energyDelta = 0

			s.log.TRACE.Printf("chargedEnergy: %.0fWh, energyDelta: %0.0fWh, energyPerSocStep: %0.0fWh, virtualBatCap: %0.1fkWh", s.chargedEnergy, energyDelta, s.energyPerSocStep, s.energyPerSocStep/10)
		}

		s.socCharge = math.Min(f+(energyDelta/s.energyPerSocStep), 100)
		s.log.TRACE.Printf("last vehicle api soc: %.2f%%, estimated soc: %.2f%%", s.prevSoC, s.socCharge)
	}

	return s.socCharge, nil
}
