package core

import (
	"slices"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
)

var demandPredictors = []string{"daily", "weekday", "temperature"}

// DefaultDemandPredictor returns the demand predictor a heating charger declares via its predictor feature
func DefaultDemandPredictor(charger any) string {
	switch {
	case hasFeature(charger, api.DemandWeekday):
		return "weekday"
	case hasFeature(charger, api.DemandTemperature):
		return "temperature"
	default:
		return "daily"
	}
}

// GetDemandPredictor returns the configured demand predictor, empty for the charger default
func (lp *Loadpoint) GetDemandPredictor() string {
	lp.RLock()
	defer lp.RUnlock()
	return lp.predictor
}

// SetDemandPredictor sets the demand predictor ("daily", "weekday", "temperature" or empty for the charger default), other values are ignored
func (lp *Loadpoint) SetDemandPredictor(predictor string) {
	if predictor != "" && !slices.Contains(demandPredictors, predictor) {
		return
	}

	lp.Lock()
	changed := lp.predictor != predictor
	lp.predictor = predictor
	lp.Unlock()

	if changed {
		lp.log.DEBUG.Printf("set demand predictor: %q", predictor)
		lp.settings.SetString(keys.DemandPredictor, predictor)
		lp.triggerOptimizer()
	}
}

func (lp *Loadpoint) activeDemandPredictor() string {
	if p := lp.GetDemandPredictor(); p != "" {
		return p
	}
	return DefaultDemandPredictor(lp.charger)
}
