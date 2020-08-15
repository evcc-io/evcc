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

	socCharge                float64 // Vehicle SoC display (estimated)
	socChargeFromAPI         float64 // Vehicle SoC read from car API
	energyPerSocStep         float64 // Energy / SOC
	chargedEnergyAtSocUpdate float64 // Charged energy at last soc update

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
	s.socChargeFromAPI = -1
	s.chargedEnergyAtSocUpdate = 0
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

// ChargeState implements Vehicle.ChargeState interface
func (s *SocEstimator) ChargeState() (float64, error) {
	f, err := s.vehicle.ChargeState()
	if err != nil {
		return s.socCharge, err
	}

	s.socCharge = f

	if s.estimate {
		socDelta := f - s.socChargeFromAPI
		energyDelta := s.chargedEnergy - s.chargedEnergyAtSocUpdate

		// soc value updated
		if socDelta > 0 || energyDelta < 0 {
			// calculate gradient, wh per soc %
			if (s.socChargeFromAPI > 0) && (socDelta >= 2) && (energyDelta > 0) {
				s.energyPerSocStep = energyDelta / socDelta
			}

			s.chargedEnergyAtSocUpdate = s.chargedEnergy
			energyDelta = 0
		}

		s.socChargeFromAPI = f
		s.socCharge = math.Min(f+(energyDelta/s.energyPerSocStep), 100)
		s.log.TRACE.Printf("chargedEnergy: %.0fWh, energyDelta: %0.0fWh, energyPerSocStep: %0.0fWh, virtualBatCap: %0.1fkWh", s.chargedEnergy, energyDelta, s.energyPerSocStep, s.energyPerSocStep/10)
		s.log.TRACE.Printf("last vehicle api soc: %.2f%%, estimated soc: %.2f%%", s.socChargeFromAPI, s.socCharge)
	}

	return s.socCharge, nil
}
