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

const (
	// homeProfileDays is the number of days the home profile is built from
	homeProfileDays = 28
	// homeProfileLookback is how far back days are searched to replace excluded days
	homeProfileLookback = 56
	// homeProfileMinDays is the minimum number of remaining days to use an excluding profile
	homeProfileMinDays = 7
)

// homeExclusionSettle delays caching after midnight until the last slot of the previous day is persisted
const homeExclusionSettle = 2 * tariff.SlotDuration

// homeExclusion holds the days excluded from the home profile, computed once per day
type homeExclusion struct {
	day     time.Time // beginning of the day the exclusion was computed for
	from    time.Time // start of the profile window
	exclude []string  // excluded local days (YYYY-MM-DD)
}

// homeProfile returns the predicted home base load in Wh for minLen 15min slots
// starting now.
func (site *Site) homeProfile(minLen int) ([]float64, error) {
	col := site.collectors[metrics.Home]

	base, err := site.homeProfileExcluding(col, time.Now())
	if err != nil {
		site.log.WARN.Printf("home profile: %v, using all days", err)
	}

	if base == nil {
		if base, err = col.EnergyProfile(now.BeginningOfDay().AddDate(0, 0, -homeProfileDays)); err != nil {
			return nil, err
		}
	}

	// convert to Wh
	return lo.Map(tileAndTrim(base[:], minLen), func(v float64, _ int) float64 { return v * 1e3 }), nil
}

// homeProfileExcluding returns the home profile without absent and outlier days, or nil
// when no day is excluded. Vacation days would otherwise bias the forecast low for weeks.
func (site *Site) homeProfileExcluding(col *metrics.Collector, ts time.Time) (*[96]float64, error) {
	ex, err := site.homeDayExclusion(col, ts)
	if err != nil || len(ex.exclude) == 0 {
		return nil, err
	}

	return col.EnergyProfileExcluding(ex.from, ex.exclude)
}

// homeDayExclusion returns the day exclusion of ts' day, cached once the previous day has settled
func (site *Site) homeDayExclusion(col *metrics.Collector, ts time.Time) (*homeExclusion, error) {
	site.homeExclusionMu.Lock()
	defer site.homeExclusionMu.Unlock()

	bod := now.With(ts).BeginningOfDay()
	if ex := site.homeExclusion; ex != nil && ex.day.Equal(bod) {
		return ex, nil
	}

	classes, err := col.ClassifyDays(bod.AddDate(0, 0, -homeProfileLookback), bod)
	if err != nil {
		return nil, err
	}

	ex := excludeDays(classes, bod)
	if ts.Sub(bod) >= homeExclusionSettle {
		site.homeExclusion = ex
	}

	if len(ex.exclude) > 0 {
		site.log.DEBUG.Printf("home profile: excluding %d absent/outlier days since %s: %v", len(ex.exclude), ex.from.Format(time.DateOnly), ex.exclude)
	}

	return ex, nil
}

// excludeDays walks back from the day before bod until homeProfileDays days with data are
// kept, at most homeProfileLookback days. Incomplete days are kept as before, days without
// data are not counted. Nothing is excluded when fewer than homeProfileMinDays days remain.
func excludeDays(classes map[string]metrics.DayClass, bod time.Time) *homeExclusion {
	res := &homeExclusion{
		day:  bod,
		from: bod.AddDate(0, 0, -homeProfileLookback),
	}

	var kept int
	for i := 1; i <= homeProfileLookback; i++ {
		day := bod.AddDate(0, 0, -i)
		key := day.Format(time.DateOnly)

		class, ok := classes[key]
		if !ok {
			continue
		}

		if class == metrics.DayAbsent || class == metrics.DayOutlier {
			res.exclude = append(res.exclude, key)
			continue
		}

		if kept++; kept == homeProfileDays {
			res.from = day
			break
		}
	}

	if kept < homeProfileMinDays {
		res.exclude = nil
	}

	return res
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
			predictorType = "daily"
			p = tileAndTrim(profile[:], minLen)
			if correct {
				predictorType = "temperature"
				p = site.applyTemperatureCorrection(p)
			}
		} else if wp := lp.demandProfileWeekday(minLen); wp != nil {
			predictorType = "weekday"
			p = wp
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

		site.log.DEBUG.Printf("heating demand: added %s forecast for %s (%d slots)",
			predictorType, lp.GetTitle(), len(p))

		res = append(res, heatingDemand{lp, p})
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
	if err != nil {
		site.log.ERROR.Printf("temperature correction: no rates available: %v", err)
		return profile
	}
	if len(rates) == 0 {
		site.log.DEBUG.Println("temperature correction: no rates available")
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
			site.log.DEBUG.Printf("temperature correction: missing historical temperature data for hour %02d:00, skipping correction", h)
			return profile
		}
	}

	res := slices.Clone(profile)
	slotStart := currentTime.Truncate(tariff.SlotDuration)
	logged := 0

	for i := range profile {
		ts := slotStart.Add(time.Duration(i) * tariff.SlotDuration)
		h := ts.UTC().Hour()

		r, err := rates.At(ts)
		if err != nil {
			site.log.DEBUG.Printf("temperature correction: slot %s: no rate available: %v", ts.Local().Format("15:04"), err)
			continue
		}
		tFuture := r.Value

		// above the heating threshold the correction is skipped, keeping the
		// historical average in the model (e.g. summer DHW still consumes energy)
		if tFuture >= heatingStopThreshold {
			if logged < 3 && profile[i] > 0 {
				site.log.DEBUG.Printf("temperature correction: slot %s (h=%02d): forecast=%.1f°C >= threshold=%.1f°C, keeping unscaled profile",
					ts.Local().Format("15:04"), h, tFuture, heatingStopThreshold)
				logged++
			}
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
			site.log.DEBUG.Printf("temperature correction: slot %s (h=%02d): forecast=%.1f°C, hist_avg=%.1f°C -> factor=%.2fx (load: %.0fWh -> %.0fWh)",
				ts.Local().Format("15:04"), h, tFuture, pastAvg, factor, profile[i]*1e3, res[i]*1e3)
			logged++
		}
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
