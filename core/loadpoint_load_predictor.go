package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
)

// demandProfile returns the heating demand profile of a heating loadpoint and whether
// it needs to be scaled by the outdoor temperature forecast. Returns nil when unavailable.
// For DemandWeekday devices, use demandProfileWeekday instead.
func (lp *Loadpoint) demandProfile() (*[96]float64, bool) {
	if lp.chargeEnergy == nil || !lp.chargerHasFeature(api.Heating) {
		return nil, false
	}

	// DemandWeekday profiles are assembled per-day in demandProfileWeekday
	if lp.chargerHasFeature(api.DemandWeekday) {
		return nil, false
	}

	temp := lp.chargerHasFeature(api.DemandTemperature)

	const minActiveHeatingEnergy = 8.0 // kWh/day threshold to classify a day as active heating

	var profile *[96]float64
	var err error

	if temp {
		// Active heating days baseline (last 7 days with >= minActiveHeatingEnergy) to prevent
		// baseline dilution from warm/idle days during seasonal transitions.
		// Falls back to standard 7-day average if insufficient active days exist.
		profile, err = lp.chargeEnergy.EnergyProfileActiveDays(7, minActiveHeatingEnergy)
		if err != nil {
			lp.log.DEBUG.Printf("active days demand profile: %v, falling back to 7-day average", err)
			from := now.BeginningOfDay().AddDate(0, 0, -7)
			profile, err = lp.chargeEnergy.EnergyProfile(from)
		}
	} else {
		from := now.BeginningOfDay().AddDate(0, 0, -28) // default: 28-day daily average
		profile, err = lp.chargeEnergy.EnergyProfile(from)
	}

	if err != nil {
		lp.log.DEBUG.Printf("demand profile: %v", err)
		return nil, false
	}

	return profile, temp
}

// demandProfileWeekday builds a minLen-slot demand forecast for a DemandWeekday device
// by fetching the correct weekday profile for each calendar day in the horizon.
func (lp *Loadpoint) demandProfileWeekday(minLen int) []float64 {
	if lp.chargeEnergy == nil || !lp.chargerHasFeature(api.DemandWeekday) {
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
