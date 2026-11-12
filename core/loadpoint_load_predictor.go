package core

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
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

// demandProfile returns the heating demand profile of a heating loadpoint and whether
// it needs to be corrected by the outdoor temperature forecast. Returns nil when unavailable.
// For weekday predictor devices, use demandProfileWeekday instead.
func (lp *Loadpoint) demandProfile() (*[96]float64, bool) {
	if lp.chargeEnergy == nil || !lp.chargerHasFeature(api.Heating) {
		return nil, false
	}

	predictor := lp.activeDemandPredictor()

	// weekday predictor profiles are assembled per-day in demandProfileWeekday
	if predictor == "weekday" {
		return nil, false
	}

	temp := predictor == "temperature"

	var from = now.BeginningOfDay().AddDate(0, 0, -28) // default: 28-day daily average
	if temp {
		from = now.BeginningOfDay().AddDate(0, 0, -7) // temperature: 7-day window
	}

	profile, err := lp.chargeEnergy.EnergyProfile(from)
	if err != nil {
		lp.log.DEBUG.Printf("demand profile: %v", err)
		return nil, false
	}

	return profile, temp
}

// demandProfileWeekday builds a minLen-slot demand forecast for a weekday predictor device
// by fetching the correct weekday profile for each calendar day in the horizon.
func (lp *Loadpoint) demandProfileWeekday(minLen int) []float64 {
	if lp.chargeEnergy == nil || !lp.chargerHasFeature(api.Heating) || lp.activeDemandPredictor() != "weekday" {
		return nil
	}

	firstSlot := int(time.Now().Truncate(tariff.SlotDuration).Sub(now.BeginningOfDay()) / tariff.SlotDuration)
	res := make([]float64, minLen)

	for i := range minLen {
		// which absolute slot within the day does index i map to?
		absSlot := firstSlot + i
		day := absSlot / 96
		slotInDay := absSlot % 96

		// fetch the weekday profile for that calendar day
		weekday := now.BeginningOfDay().AddDate(0, 0, day).Weekday()
		profile, err := lp.chargeEnergy.EnergyProfileWeekday(weekday)
		if err != nil {
			lp.log.DEBUG.Printf("demand profile weekday (%s): %v", weekday, err)
			return nil
		}

		res[i] = profile[slotInDay]
	}

	return res
}
