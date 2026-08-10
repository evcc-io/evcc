package core

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
)

// homeProfile returns the predicted home load in Wh for minLen 15min slots starting now.
// Base load uses same-weekday avg (past 4 weeks). Heating loadpoints add their
// demand profile on top: DemandProfileDailyTemperature (temp-scaled) or
// DemandProfileSameWeekday (same-weekday avg).
func (site *Site) homeProfile(minLen int) ([]float64, error) {
	// base load (excludes loadpoints) - avg of same weekday over past 4 weeks
	gt_base, err := site.collectors[metrics.Home].EnergyProfileWeekday()
	if err != nil {
		return nil, err
	}

	// tile base profile to cover minLen slots (allow for prorating first day)
	slots := make([]float64, 0, minLen+1)
	for len(slots) <= minLen+24*4 {
		slots = append(slots, gt_base[:]...)
	}

	res := profileSlotsFromNow(slots)
	if len(res) < minLen {
		return nil, fmt.Errorf("minimum home profile length %d is less than required %d", len(res), minLen)
	}
	if len(res) > minLen {
		res = res[:minLen]
	}

	tempProfile, sameWeekdayProfile := site.extractHeaterProfiles()

	if len(tempProfile) == 0 && len(sameWeekdayProfile) == 0 {
		site.log.DEBUG.Println("home profile: no heating devices, returning base load only")
		return lo.Map(res, func(v float64, i int) float64 { return v * 1e3 }), nil
	}

	gt_final := make([]float64, len(res))
	copy(gt_final, res)

	// DemandProfileDailyTemperature: daily-averaged profile scaled by outdoor temp forecast
	if len(tempProfile) > 0 {
		p := tileAndTrim(tempProfile, minLen)
		site.log.DEBUG.Printf("home profile: applying temperature correction to %d slots", len(p))
		p = site.applyTemperatureCorrection(p)
		for i := range gt_final {
			if i < len(p) {
				gt_final[i] += p[i]
			}
		}
	}

	// DemandProfileSameWeekday: avg of same weekday over past 4 weeks, tiled as-is
	if len(sameWeekdayProfile) > 0 {
		p := tileAndTrim(sameWeekdayProfile, minLen)
		site.log.DEBUG.Printf("home profile: adding same-weekday demand profile with %d slots", len(p))
		for i := range gt_final {
			if i < len(p) {
				gt_final[i] += p[i]
			}
		}
	}

	// convert to Wh
	return lo.Map(gt_final, func(v float64, i int) float64 { return v * 1e3 }), nil
}

// extractHeaterProfiles returns aggregated per-strategy heating profiles.
// tempProfile: loadpoints with DemandProfileDailyTemperature (daily avg, scaled by outdoor temp).
// sameWeekdayProfile: loadpoints with DemandProfileSameWeekday (avg of same weekday, past 4 weeks).
func (site *Site) extractHeaterProfiles() (tempProfile, sameWeekdayProfile []float64) {
	var tempProfiles, sameWeekdayProfiles [][]float64

	for i, lp := range site.loadpoints {
		if lp.chargeEnergy == nil || !hasFeature(lp.charger, api.Heating) {
			continue
		}

		if hasFeature(lp.charger, api.DemandProfileDailyTemperature) {
			profile, err := lp.chargeEnergy.EnergyProfile(now.BeginningOfDay().AddDate(0, 0, -7))
			switch {
			case err == nil:
				site.log.DEBUG.Printf("heater profile: loadpoint %d: temperature strategy, %d slots", i, len(profile))
				tempProfiles = append(tempProfiles, profile[:])
			case errors.Is(err, metrics.ErrIncomplete):
				site.log.DEBUG.Printf("heater profile: loadpoint %d: temperature strategy, insufficient data", i)
			default:
				site.log.DEBUG.Printf("heater profile: loadpoint %d: temperature strategy, no data (%v)", i, err)
			}
		} else if hasFeature(lp.charger, api.DemandProfileSameWeekday) {
			profile, err := lp.chargeEnergy.EnergyProfileWeekday()
			switch {
			case err == nil:
				site.log.DEBUG.Printf("heater profile: loadpoint %d: same-weekday strategy, %d slots", i, len(profile))
				sameWeekdayProfiles = append(sameWeekdayProfiles, profile[:])
			case errors.Is(err, metrics.ErrIncomplete):
				site.log.DEBUG.Printf("heater profile: loadpoint %d: same-weekday strategy, insufficient data", i)
			default:
				site.log.DEBUG.Printf("heater profile: loadpoint %d: same-weekday strategy, no data (%v)", i, err)
			}
		}
	}

	if len(tempProfiles) > 0 {
		tempProfile = sumProfiles(tempProfiles)
		site.log.DEBUG.Printf("heater profile: aggregated %d temperature loadpoint(s)", len(tempProfiles))
	}
	if len(sameWeekdayProfiles) > 0 {
		sameWeekdayProfile = sumProfiles(sameWeekdayProfiles)
		site.log.DEBUG.Printf("heater profile: aggregated %d same-weekday loadpoint(s)", len(sameWeekdayProfiles))
	}

	return tempProfile, sameWeekdayProfile
}

// applyTemperatureCorrection adjusts heating load based on temperature forecast.
// Uses formula: load[i] = load_avg[i] × ((T_room − T_forecast[i]) / (T_room − T_past_avg[h]))
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
	)

	currentTime := time.Now()

	// compute average historical temperature per hour-of-day
	pastTempSum := make([]float64, 24)
	pastTempCount := make([]int, 24)
	for _, r := range rates {
		if r.Start.Before(currentTime) {
			h := r.Start.UTC().Hour()
			pastTempSum[h] += r.Value
			pastTempCount[h]++
		}
	}
	pastTempAvg := make([]float64, 24)
	for h := range 24 {
		if pastTempCount[h] > 0 {
			pastTempAvg[h] = pastTempSum[h] / float64(pastTempCount[h])
		}
	}

	ratesByTime := make(map[time.Time]float64, len(rates))
	for _, r := range rates {
		ratesByTime[r.Start] = r.Value
	}

	result := make([]float64, len(profile))
	copy(result, profile)

	slotStart := currentTime.Truncate(tariff.SlotDuration)
	for i := range profile {
		ts := slotStart.Add(time.Duration(i) * tariff.SlotDuration)

		tFuture, found := ratesByTime[ts]
		if !found {
			continue
		}

		h := ts.UTC().Hour()

		if pastTempCount[h] == 0 {
			site.log.DEBUG.Printf("temperature correction: no historical data for hour %d, skipping slot %s", h, ts.Format("15:04"))
			continue
		}

		tPastAvg := pastTempAvg[h]

		denominator := tRoom - tPastAvg
		numerator := tRoom - tFuture

		if math.Abs(denominator) < 0.5 || tFuture >= heatingStopThreshold {
			continue
		}

		correctionFactor := numerator / denominator

		// Clamp to reasonable range to prevent extreme corrections from bad data
		const minCorrectionFactor = 0.5 // -50% (warmer than expected)
		const maxCorrectionFactor = 2.0 // +100% (colder than expected)
		correctionFactor = math.Max(minCorrectionFactor, math.Min(maxCorrectionFactor, correctionFactor))

		oldValue := profile[i]
		result[i] = oldValue * correctionFactor

		if i < 3 && oldValue != 0 {
			site.log.DEBUG.Printf("temperature correction sample: slot %s (hour %d): forecast=%.1f°C, hist_avg=%.1f°C, factor=%.3f, load: %.0fWh -> %.0fWh (%.1f%%)",
				ts.Format("15:04"), h, tFuture, tPastAvg, correctionFactor, oldValue*1e3, result[i]*1e3, ((result[i]/oldValue)-1)*100)
		}
	}

	return result
}

// tileAndTrim repeats profile until it covers minLen slots, then trims and aligns to now.
func tileAndTrim(profile []float64, minLen int) []float64 {
	slots := make([]float64, 0, minLen+1)
	for len(slots) <= minLen+24*4 {
		slots = append(slots, profile...)
	}
	res := profileSlotsFromNow(slots)
	if len(res) > minLen {
		res = res[:minLen]
	}
	return res
}

// sumProfiles element-wise sums a slice of equal-length profiles.
func sumProfiles(profiles [][]float64) []float64 {
	if len(profiles) == 0 {
		return nil
	}

	result := make([]float64, len(profiles[0]))
	for _, profile := range profiles {
		for i := 0; i < len(result) && i < len(profile); i++ {
			result[i] += profile[i]
		}
	}
	return result
}

// profileSlotsFromNow strips slots before now, aligning the profile to the current 15min slot.
func profileSlotsFromNow(profile []float64) []float64 {
	firstSlot := int(time.Now().Truncate(tariff.SlotDuration).Sub(now.BeginningOfDay()) / tariff.SlotDuration)
	return profile[firstSlot:]
}
