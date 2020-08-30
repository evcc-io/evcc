package charger

// This file has been generated - do not modify

import (
	"github.com/andig/evcc/api"
)

func decorateCharger(base api.Charger, chargePhases func(int64) error) api.Charger {
	switch {
	case chargePhases == nil:
		return base

	case chargePhases != nil:
		return &struct {
			api.Charger
			api.ChargePhases
		}{
			Charger: base,
			ChargePhases: &decorateChargerChargePhasesImpl{
				chargePhases: chargePhases,
			},
		}
	}

	return nil
}

type decorateChargerChargePhasesImpl struct {
	chargePhases func(int64) error
}

func (impl *decorateChargerChargePhasesImpl) Phases1p3p(phases int64) error {
	return impl.chargePhases(phases)
}
