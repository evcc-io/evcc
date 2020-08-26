package charger

import (
	"github.com/andig/evcc/api"
)

func decorateOpenWB(base api.Charger, chargePhases func(int64) error) api.Charger {
	switch {
	case chargePhases == nil:
		return base

	case chargePhases != nil:
		return &struct {
			api.Charger
			api.ChargePhases
		}{
			Charger: base,
			ChargePhases: &decorateOpenWBChargePhasesImpl{
				chargePhases: chargePhases,
			},
		}
	}

	return nil
}

type decorateOpenWBChargePhasesImpl struct {
	chargePhases func(int64) error
}

func (impl *decorateOpenWBChargePhasesImpl) Phases1p3p(phases int64) error {
	return impl.chargePhases(phases)
}
