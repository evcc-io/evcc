package core

import (
	"time"

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

	temp := lp.chargerHasFeature(api.DemandTemperature)

	switch {
	case temp:
		profile, err = lp.chargeEnergy.EnergyProfile(now.BeginningOfDay().AddDate(0, 0, -7))

	case lp.chargerHasFeature(api.DemandDaily):
		profile, err = lp.chargeEnergy.EnergyProfile(now.BeginningOfDay().AddDate(0, 0, -28))

	case lp.chargerHasFeature(api.DemandWeekday):
		profile, err = lp.chargeEnergy.EnergyProfileWeekday(time.Now().Weekday())

	default:
		return nil, false
	}

	if err != nil {
		lp.log.DEBUG.Printf("demand profile: %v", err)
		return nil, false
	}

	return profile, temp
}
