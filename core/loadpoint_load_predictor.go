package core

import (
	"github.com/evcc-io/evcc/api"
	"github.com/jinzhu/now"
)

// demandProfile returns the heating demand profile of a heating loadpoint and whether
// it needs to be scaled by the outdoor temperature forecast. Returns nil when unavailable.
func (lp *Loadpoint) demandProfile() (*[96]float64, bool) {
	if lp.chargeEnergy == nil || !lp.chargerHasFeature(api.Heating) {
		return nil, false
	}

	var profile *[96]float64
	var err error

	// daily avg scaled by outdoor temp
	correct := lp.chargerHasFeature(api.DemandProfileDailyTemperature)
	from := now.BeginningOfDay().AddDate(0, 0, -28)

	switch {
	case correct:
		profile, err = lp.chargeEnergy.EnergyProfile(from)

	// avg of same weekday over past 4 weeks, used as-is
	case lp.chargerHasFeature(api.DemandProfileSameWeekday):
		profile, err = lp.chargeEnergy.EnergyProfileWeekday(from)

	default:
		return nil, false
	}

	if err != nil {
		lp.log.DEBUG.Printf("demand profile: %v", err)
		return nil, false
	}

	return profile, correct
}
