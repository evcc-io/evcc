package core

import (
	"fmt"
	"math"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
)

// homeProfile returns the predicted home load in Wh for minLen 15min slots starting now.
// Base load is the same-weekday avg (past 4 weeks), heating loadpoints add their demand profile on top.
func (site *Site) homeProfile(minLen int) ([]float64, error) {
	// base load (excludes loadpoints) - avg of same weekday over past 4 weeks
	base, err := site.collectors[metrics.Home].EnergyProfileWeekday()
	if err != nil {
		return nil, err
	}

	res := tileAndTrim(base[:], minLen)
	if len(res) < minLen {
		return nil, fmt.Errorf("minimum home profile length %d is less than required %d", len(res), minLen)
	}

	tempProfile, sameWeekdayProfile := site.extractHeaterProfiles()

	// DemandProfileDailyTemperature: daily-averaged profile scaled by outdoor temp forecast
	if len(tempProfile) > 0 {
		addProfile(res, site.applyTemperatureCorrection(tileAndTrim(tempProfile, minLen)))
	}

	// DemandProfileSameWeekday: avg of same weekday over past 4 weeks, tiled as-is
	if len(sameWeekdayProfile) > 0 {
		addProfile(res, tileAndTrim(sameWeekdayProfile, minLen))
	}

	// convert to Wh
	return lo.Map(res, func(v float64, _ int) float64 { return v * 1e3 }), nil
}

// extractHeaterProfiles returns the aggregated heating profiles of all heating
// loadpoints, split by their demand profile strategy.
func (site *Site) extractHeaterProfiles() (tempProfile, sameWeekdayProfile []float64) {
	for i, lp := range site.loadpoints {
		if lp.chargeEnergy == nil || !hasFeature(lp.charger, api.Heating) {
			continue
		}

		target := &sameWeekdayProfile
		fetch := lp.chargeEnergy.EnergyProfileWeekday

		switch {
		case hasFeature(lp.charger, api.DemandProfileDailyTemperature):
			target = &tempProfile
			fetch = func() (*[96]float64, error) {
				return lp.chargeEnergy.EnergyProfile(now.BeginningOfDay().AddDate(0, 0, -7))
			}
		case !hasFeature(lp.charger, api.DemandProfileSameWeekday):
			continue
		}

		profile, err := fetch()
		if err != nil {
			site.log.DEBUG.Printf("heater profile: loadpoint %d: %v", i, err)
			continue
		}

		if *target == nil {
			*target = make([]float64, len(profile))
		}
		addProfile(*target, profile[:])
	}

	return tempProfile, sameWeekdayProfile
}

// applyTemperatureCorrection adjusts heating load based on temperature forecast:
// load[i] = load_avg[i] × ((T_room − T_forecast[i]) / (T_room − T_past_avg[h]))
func (site *Site) applyTemperatureCorrection(profile []float64) []float64 {
	weatherTariff := site.GetTariff(api.TariffUsageTemperature)
	if weatherTariff == nil {
		return profile
	}

	rates, err := weatherTariff.Rates()
	if err != nil || len(rates) == 0 {
		return profile
	}

	const (
		tRoom                = 21.0
		heatingStopThreshold = 18.0
		minCorrection        = 0.5 // warmer than expected
		maxCorrection        = 2.0 // colder than expected
	)

	currentTime := time.Now()

	// average historical temperature per hour-of-day
	var pastSum [24]float64
	var pastCount [24]int
	forecast := make(map[time.Time]float64, len(rates))

	for _, r := range rates {
		forecast[r.Start] = r.Value

		if r.Start.Before(currentTime) {
			h := r.Start.UTC().Hour()
			pastSum[h] += r.Value
			pastCount[h]++
		}
	}

	res := slices.Clone(profile)
	slotStart := currentTime.Truncate(tariff.SlotDuration)

	for i := range profile {
		ts := slotStart.Add(time.Duration(i) * tariff.SlotDuration)
		h := ts.UTC().Hour()

		tFuture, ok := forecast[ts]
		if !ok || pastCount[h] == 0 || tFuture >= heatingStopThreshold {
			continue
		}

		denominator := tRoom - pastSum[h]/float64(pastCount[h])
		if math.Abs(denominator) < 0.5 {
			continue
		}

		// clamp to prevent extreme corrections from bad data
		res[i] = profile[i] * min(maxCorrection, max(minCorrection, (tRoom-tFuture)/denominator))
	}

	return res
}

// tileAndTrim repeats profile until it covers minLen slots, then trims and aligns to now.
func tileAndTrim(profile []float64, minLen int) []float64 {
	slots := make([]float64, 0, minLen+1)
	for len(slots) <= minLen+24*4 { // allow for prorating first day
		slots = append(slots, profile...)
	}

	res := profileSlotsFromNow(slots)
	if len(res) > minLen {
		res = res[:minLen]
	}

	return res
}

// addProfile adds src to dst element-wise, limited to the shorter of both.
func addProfile(dst, src []float64) {
	for i := range min(len(dst), len(src)) {
		dst[i] += src[i]
	}
}

// profileSlotsFromNow strips slots before now, aligning the profile to the current 15min slot.
func profileSlotsFromNow(profile []float64) []float64 {
	firstSlot := int(time.Now().Truncate(tariff.SlotDuration).Sub(now.BeginningOfDay()) / tariff.SlotDuration)
	return profile[firstSlot:]
}
