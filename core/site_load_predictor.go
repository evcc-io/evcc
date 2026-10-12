package core

import (
	"errors"
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

// heatingDemand is the forecast demand in Wh a single heating loadpoint contributes
// to the home load
type heatingDemand struct {
	lp     loadpoint.API
	values []float64
}

// addHeatingDemand adds the forecast demand of all heating loadpoints to the home load
// in Wh and returns the contributions. Must be applied after blending the measured home
// energy, which does not contain loadpoint power.
func (site *Site) addHeatingDemand(gt []float64, minLen int) []heatingDemand {
	var res []heatingDemand

	for _, lp := range site.loadpoints {
		if lp == nil {
			continue
		}

		// skip disconnected or disabled heating loadpoints: the historical profile
		// must not be added for consumption that will not occur
		if s := lp.GetStatus(); s != api.StatusB && s != api.StatusC {
			continue
		}
		// For Continuous devices ModeOff means "normal operation" (heat pump runs
		// on its own schedule). Skip only non-continuous loadpoints in ModeOff.
		if lp.GetMode() == api.ModeOff && !lp.chargerHasFeature(api.Continuous) {
			continue
		}

		var p []float64
		var predictorType string

		if profile, correct := lp.demandProfile(); profile != nil {
			p = tileAndTrim(profile[:], minLen)
			if correct {
				var binned map[int]map[int]float64
				var err error
				binned, err = lp.chargeEnergy.EnergyProfileTemperatureBinned()
				if err != nil && !errors.Is(err, metrics.ErrNoBinnedData) {
					site.log.DEBUG.Printf("temperature correction: binned profile unavailable for %s: %v", lp.GetTitle(), err)
				}
				p = site.applyTemperatureCorrection(lp.GetTitle(), p, binned)
			} else {
				site.log.DEBUG.Printf("heating demand: added daily forecast for %s (%d slots)",
					lp.GetTitle(), len(p))
			}
		} else if wp := lp.demandProfileWeekday(minLen); wp != nil {
			predictorType = "weekday"
			p = wp
			site.log.DEBUG.Printf("heating demand: added %s forecast for %s (%d slots)",
				predictorType, lp.GetTitle(), len(p))
		} else {
			continue
		}

		// profiles are kWh
		for i := range p {
			p[i] *= 1e3
		}

		for i := range min(len(gt), len(p)) {
			gt[i] += p[i]
		}

		res = append(res, heatingDemand{lp, p})
	}

	return res
}

// lookupBin returns the historical energy for the given forecast temperature (°C)
// by checking the exact rounded bin first, then falling back to the average of adjacent ±1°C bins.
func lookupBin(bins map[int]float64, t float64) (float64, bool) {
	if bins == nil {
		return 0, false
	}

	tempBin := int(math.Round(t))

	if val, ok := bins[tempBin]; ok {
		return val, true
	}

	// Check adjacent ±1°C bins
	valMinus, okMinus := bins[tempBin-1]
	valPlus, okPlus := bins[tempBin+1]
	if okMinus && okPlus {
		return (valMinus + valPlus) / 2, true
	}
	if okMinus {
		return valMinus, true
	}
	if okPlus {
		return valPlus, true
	}

	return 0, false
}

// applyTemperatureCorrection adjusts heating load based on temperature forecast:
// It looks up historical consumption for matching temperature bins (±1.5°C), and falls back
// to linear scaling when historical data is unavailable for a given slot.
func (site *Site) applyTemperatureCorrection(lpTitle string, profile []float64, binned map[int]map[int]float64) []float64 {
	weatherTariff := site.GetTariff(api.TariffUsageTemperature)
	if weatherTariff == nil {
		site.log.WARN.Println("temperature correction: demandtemperature predictor set but no temperature tariff configured")
		return profile
	}

	rates, err := weatherTariff.Rates()
	if err != nil {
		site.log.ERROR.Printf("temperature correction: no rates available: %v", err)
		return profile
	}
	if len(rates) == 0 {
		site.log.DEBUG.Println("temperature correction: no rates available")
		return profile
	}

	const (
		tRoom         = 21.0
		minCorrection = 0.1 // warmer than expected: 10% floor for standby and DHW
		maxCorrection = 3.0 // colder than expected: caps aggressive multipliers on near-zero baselines
	)

	currentTime := time.Now()

	// average historical temperature per hour-of-day (used for linear scaling fallback)
	var pastSum [24]float64
	var pastCount [24]int

	for _, r := range rates {
		if r.Start.Before(currentTime) {
			h := r.Start.UTC().Hour()
			pastSum[h] += r.Value
			pastCount[h]++
		}
	}

	res := slices.Clone(profile)
	slotStart := currentTime.Truncate(tariff.SlotDuration)
	var binnedSlots, scaledSlots, logged int

	for i := range profile {
		ts := slotStart.Add(time.Duration(i) * tariff.SlotDuration)
		h := ts.UTC().Hour()
		slotInDay := (ts.Local().Hour()*60 + ts.Local().Minute()) / 15

		r, err := rates.At(ts)
		if err != nil {
			site.log.DEBUG.Printf("temperature correction: slot %s: no rate available: %v", ts.Local().Format("15:04"), err)
			continue
		}
		tFuture := r.Value

		// 1. Empirical Temperature Binning (k-NN lookup at forecast temperature):
		// matches recorded historical consumption per time-of-day slot and ±1°C temperature bin,
		// capturing non-linear heat pump COP, defrost cycles, and scheduled DHW runs.
		if binned != nil && binned[slotInDay] != nil {
			if binnedVal, ok := lookupBin(binned[slotInDay], tFuture); ok {
				res[i] = binnedVal
				binnedSlots++
				if logged < 3 && binnedVal > 0 {
					site.log.DEBUG.Printf("temperature correction: slot %s (h=%02d): forecast=%.1f°C -> binned match [%d°C]=%.0fWh",
						ts.Local().Format("15:04"), h, tFuture, int(math.Round(tFuture)), binnedVal*1e3)
					logged++
				}
				continue
			}
		}

		// 2. Fallback: Linear Scaling based on standard building physics (DIN EN 12831 / degree-day),
		// where building heat loss is proportional to indoor/outdoor temperature difference:
		// factor = (T_room - T_forecast) / (T_room - T_past_avg)
		scaledSlots++

		// require past temperature data for hour h to compute pastAvg and avoid division by zero
		// (e.g. after startup when the weather source only provides forecast data from today onward)
		if pastCount[h] == 0 {
			site.log.DEBUG.Printf("temperature correction: slot %s (h=%02d): no historical temperature data, leaving slot unscaled",
				ts.Local().Format("15:04"), h)
			continue
		}

		pastAvg := pastSum[h] / float64(pastCount[h])
		denominator := tRoom - pastAvg
		if denominator <= 0.5 {
			site.log.DEBUG.Printf("temperature correction: slot %s (h=%02d): hist_avg=%.1f°C too close to room temp=%.1f°C, skipping slot",
				ts.Local().Format("15:04"), h, pastAvg, tRoom)
			continue
		}

		// clamp to prevent extreme corrections from bad data
		factor := min(maxCorrection, max(minCorrection, (tRoom-tFuture)/denominator))
		res[i] = profile[i] * factor

		if logged < 3 && factor != 1.0 && profile[i] > 0 {
			site.log.DEBUG.Printf("temperature correction: slot %s (h=%02d): forecast=%.1f°C, hist_avg=%.1f°C -> scaled factor=%.2fx (load: %.0fWh -> %.0fWh)",
				ts.Local().Format("15:04"), h, tFuture, pastAvg, factor, profile[i]*1e3, res[i]*1e3)
			logged++
		}
	}

	site.log.DEBUG.Printf("heating demand: added temperature forecast for %s (%d slots: %d binned, %d scaled)",
		lpTitle, len(res), binnedSlots, scaledSlots)

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
