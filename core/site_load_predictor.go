package core

import (
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/tariff"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
)

// homeProfile returns the predicted home base load in Wh for minLen 15min slots
// starting now.
func (site *Site) homeProfile(minLen int) ([]float64, error) {
	col := site.collectors[metrics.Home]

	base, err := col.EnergyProfile(now.BeginningOfDay().AddDate(0, 0, -28))
	if err != nil {
		return nil, err
	}

	// convert to Wh
	return lo.Map(tileAndTrim(base[:], minLen), func(v float64, _ int) float64 { return v * 1e3 }), nil
}

// addHeatingDemand adds the forecast demand of all heating loadpoints to the home load
// in Wh and returns the loadpoints that contributed. Must be applied after blending the
// measured home energy, which does not contain loadpoint power.
func (site *Site) addHeatingDemand(gt []float64, minLen int) []loadpoint.API {
	var res []loadpoint.API

	for _, lp := range site.loadpoints {
		if lp == nil {
			continue
		}

		// skip disconnected or disabled heating loadpoints: the historical profile
		// must not be added for consumption that will not occur
		if s := lp.GetStatus(); s != api.StatusB && s != api.StatusC {
			continue
		}
		if lp.GetMode() == api.ModeOff {
			continue
		}

		var p []float64

		if profile, correct := lp.demandProfile(); profile != nil {
			p = tileAndTrim(profile[:], minLen)
			if correct {
				p = site.applyTemperatureCorrection(p)
			}
		} else if wp := lp.demandProfileWeekday(minLen); wp != nil {
			p = wp
		} else {
			continue
		}

		for i := range min(len(gt), len(p)) {
			gt[i] += p[i] * 1e3
		}

		res = append(res, lp)
	}

	return res
}

// applyTemperatureCorrection adjusts heating load based on temperature forecast:
// load[i] = load_avg[i] × ((T_room − T_forecast[i]) / (T_room − T_past_avg[h]))
func (site *Site) applyTemperatureCorrection(profile []float64) []float64 {
	weatherTariff := site.GetTariff(api.TariffUsageTemperature)
	if weatherTariff == nil {
		site.log.WARN.Println("temperature correction: demandtemperature predictor set but no temperature tariff configured")
		return profile
	}

	rates, err := weatherTariff.Rates()
	if err != nil || len(rates) == 0 {
		site.log.ERROR.Printf("temperature correction: no rates available: %v", err)
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

	for _, r := range rates {
		if r.Start.Before(currentTime) {
			h := r.Start.UTC().Hour()
			pastSum[h] += r.Value
			pastCount[h]++
		}
	}

	// require at least one past sample for every hour-of-day; otherwise the
	// correction would be applied to some slots but not others, mixing two
	// models in the same profile. This is common after a restart when the
	// temperature source only provides data from today onward.
	for h := range pastCount {
		if pastCount[h] == 0 {
			return profile
		}
	}

	res := slices.Clone(profile)
	slotStart := currentTime.Truncate(tariff.SlotDuration)

	for i := range profile {
		ts := slotStart.Add(time.Duration(i) * tariff.SlotDuration)
		h := ts.UTC().Hour()

		r, err := rates.At(ts)
		if err != nil {
			continue
		}
		tFuture := r.Value

		// above the heating threshold the correction is skipped, keeping the
		// historical average in the model (e.g. summer DHW still consumes energy)
		if tFuture >= heatingStopThreshold {
			continue
		}

		denominator := tRoom - pastSum[h]/float64(pastCount[h])
		if denominator <= 0.5 {
			continue
		}

		// clamp to prevent extreme corrections from bad data
		res[i] = profile[i] * min(maxCorrection, max(minCorrection, (tRoom-tFuture)/denominator))
	}

	return res
}

// tileAndTrim returns minLen slots of the repeating daily profile, starting at the current 15min slot.
func tileAndTrim(profile []float64, minLen int) []float64 {
	firstSlot := int(time.Now().Truncate(tariff.SlotDuration).Sub(now.BeginningOfDay()) / tariff.SlotDuration)

	res := make([]float64, minLen)
	for i := range res {
		res[i] = profile[(firstSlot+i)%len(profile)]
	}

	return res
}
