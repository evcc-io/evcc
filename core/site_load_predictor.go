package core

import (
	"math"
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

		profile, correct := lp.demandProfile()
		if profile == nil {
			continue
		}

		p := tileAndTrim(profile[:], minLen)
		if correct {
			p = site.applyTemperatureCorrection(p)
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
		if !ok || pastCount[h] == 0 {
			continue
		}

		// heating stops once it is warm enough outside
		if tFuture >= heatingStopThreshold {
			res[i] = 0
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

// tileAndTrim returns minLen slots of the repeating daily profile, starting at the current 15min slot.
func tileAndTrim(profile []float64, minLen int) []float64 {
	firstSlot := int(time.Now().Truncate(tariff.SlotDuration).Sub(now.BeginningOfDay()) / tariff.SlotDuration)

	res := make([]float64, minLen)
	for i := range res {
		res[i] = profile[(firstSlot+i)%len(profile)]
	}

	return res
}
