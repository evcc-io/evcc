package core

import (
	"encoding/json"
	"math"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
)

// forecastSeries and solarDetails implement BytesMarshaler so MQTT publishes one
// json message per forecast key instead of decomposing every slot into its own
// topic (several thousand messages per update).
type forecastSeries [][]float64

var _ api.BytesMarshaler = (*forecastSeries)(nil)

func (s forecastSeries) MarshalBytes() ([]byte, error) {
	return json.Marshal(s)
}

type solarDetails struct {
	Scale            float64      `json:"scale"`                // trailing percentile solar scale factor, 1 if unscaled
	Today            dailyDetails `json:"today"`                // tomorrow
	Tomorrow         dailyDetails `json:"tomorrow"`             // tomorrow
	DayAfterTomorrow dailyDetails `json:"dayAfterTomorrow"`     // day after tomorrow
	Timeseries       timeseries   `json:"timeseries,omitempty"` // timeseries of forecasted energy
}

var _ api.BytesMarshaler = (*solarDetails)(nil)

func (d solarDetails) MarshalBytes() ([]byte, error) {
	return json.Marshal(d)
}

type dailyDetails struct {
	Yield    float64 `json:"energy"`
	Complete bool    `json:"complete"`
}

// forecastRates publishes rates as [start, end, value] with the timestamps in
// unix seconds. The forecast is the largest payload evcc sends and RFC3339
// timestamps are two thirds of it.
func forecastRates(rr api.Rates) forecastSeries {
	// keep nil for empty rates: shards are published without omitempty
	if len(rr) == 0 {
		return nil
	}

	return lo.Map(rr, func(r api.Rate, _ int) []float64 {
		return []float64{float64(r.Start.Unix()), float64(r.End.Unix()), r.Value}
	})
}

// greenShare returns
//   - the current green share, calculated for the part of the consumption between powerFrom and powerTo
//     the consumption below powerFrom will get the available green power first
func (site *Site) greenShare(powerFrom float64, powerTo float64) float64 {
	greenPower := math.Max(0, site.pvPower) + math.Max(0, site.battery.Power)
	greenPowerAvailable := math.Max(0, greenPower-powerFrom)

	power := powerTo - powerFrom
	share := math.Min(greenPowerAvailable, power) / power

	if math.IsNaN(share) {
		if greenPowerAvailable > 0 {
			share = 1
		} else {
			share = 0
		}
	}

	return share
}

// effectivePrice calculates the real energy price based on self-produced and grid-imported energy.
func (site *Site) effectivePrice(greenShare float64) *float64 {
	if grid, err := tariff.Now(site.GetTariff(api.TariffUsageGrid)); err == nil {
		feedin, err := tariff.Now(site.GetTariff(api.TariffUsageFeedIn))
		if err != nil {
			feedin = 0
		}
		effPrice := grid*(1-greenShare) + feedin*greenShare
		return &effPrice
	}
	return nil
}

// effectiveCo2 calculates the amount of emitted co2 based on self-produced and grid-imported energy.
func (site *Site) effectiveCo2(greenShare float64) *float64 {
	if co2, err := tariff.Now(site.GetTariff(api.TariffUsageCo2)); err == nil {
		effCo2 := co2 * (1 - greenShare)
		return &effCo2
	}
	return nil
}

func (site *Site) publishTariffs(greenShareHome float64, greenShareLoadpoints float64) {
	site.publish(keys.GreenShareHome, greenShareHome)
	site.publish(keys.GreenShareLoadpoints, greenShareLoadpoints)

	if v, err := tariff.Now(site.GetTariff(api.TariffUsageGrid)); err == nil {
		site.publish(keys.TariffGrid, v)
	}
	if v, err := tariff.Now(site.GetTariff(api.TariffUsageFeedIn)); err == nil {
		site.publish(keys.TariffFeedIn, v)
	}
	if v, err := tariff.Now(site.GetTariff(api.TariffUsageCo2)); err == nil {
		site.publish(keys.TariffCo2, v)
	}
	if v, err := tariff.Now(site.GetTariff(api.TariffUsageSolar)); err == nil {
		site.publish(keys.TariffSolar, v)
	}
	if v, err := tariff.Now(site.GetTariff(api.TariffUsageTemperature)); err == nil {
		site.publish(keys.TariffTemperature, v)
	}
	if v := site.effectivePrice(greenShareHome); v != nil {
		site.publish(keys.TariffPriceHome, v)
	}
	if v := site.effectiveCo2(greenShareHome); v != nil {
		site.publish(keys.TariffCo2Home, v)
	}
	if v := site.effectivePrice(greenShareLoadpoints); v != nil {
		site.publish(keys.TariffPriceLoadpoints, v)
	}
	if v := site.effectiveCo2(greenShareLoadpoints); v != nil {
		site.publish(keys.TariffCo2Loadpoints, v)
	}

	fc := struct {
		Co2         forecastSeries `json:"co2,omitempty"`
		FeedIn      forecastSeries `json:"feedin,omitempty"`
		Grid        forecastSeries `json:"grid,omitempty"`
		Planner     forecastSeries `json:"planner,omitempty"`
		Solar       *solarDetails  `json:"solar,omitempty"`
		Temperature forecastSeries `json:"temperature,omitempty"`
	}{
		Co2:         forecastRates(tariff.Rates(site.GetTariff(api.TariffUsageCo2))),
		FeedIn:      forecastRates(tariff.Rates(site.GetTariff(api.TariffUsageFeedIn))),
		Planner:     forecastRates(tariff.Rates(site.GetTariff(api.TariffUsagePlanner))),
		Grid:        forecastRates(tariff.Rates(site.GetTariff(api.TariffUsageGrid))),
		Temperature: forecastRates(tariff.Rates(site.GetTariff(api.TariffUsageTemperature))),
	}

	// calculate adjusted solar rates
	if solar := tariff.Rates(site.GetTariff(api.TariffUsageSolar)); len(solar) > 0 {
		fc.Solar = new(site.solarDetails(solar))
	}

	site.publish(keys.Forecast, util.NewSharder(keys.Forecast, fc))

	site.persistTariffs()
}

// persistTariffs stores tariff values once per 15min boundary. Like the meter
// collectors it is driven by the update loop, skipping the partial boot slot.
func (site *Site) persistTariffs() {
	slot := time.Now().Truncate(tariff.SlotDuration)

	last := site.tariffSlot
	site.tariffSlot = slot

	// skip repeat ticks within the slot and the partial boot slot
	if last.IsZero() || !slot.After(last) {
		return
	}

	value := func(u api.TariffUsage) *float64 {
		if r, err := tariff.At(site.GetTariff(u), slot); err == nil {
			return &r.Value
		}
		return nil
	}

	if err := metrics.PersistTariffs(slot,
		value(api.TariffUsageGrid),
		value(api.TariffUsageFeedIn),
		value(api.TariffUsageCo2),
		value(api.TariffUsageTemperature),
	); err != nil {
		site.log.ERROR.Printf("persist tariffs: %v", err)
	}
}

// forecastSlotEnergy is the energy expected in the slot covering now, integrated
// the same way as the published forecast so the persisted history matches the
// curve the UI draws. Beyond the forecast horizon it is zero.
func forecastSlotEnergy(solar api.Rates, now time.Time) float64 {
	slot := now.Truncate(tariff.SlotDuration)
	return solarEnergy(solar, slot, slot.Add(tariff.SlotDuration)) / 1e3
}

func (site *Site) solarDetails(solar api.Rates) solarDetails {
	res := solarDetails{
		Timeseries: solarTimeseries(solar),
	}

	last := solar[len(solar)-1].Start

	bod := now.BeginningOfDay()
	eod := bod.AddDate(0, 0, 1)
	eot := eod.AddDate(0, 0, 1)

	remainingToday := solarEnergy(solar, time.Now(), eod)
	tomorrow := solarEnergy(solar, eod, eot)
	dayAfterTomorrow := solarEnergy(solar, eot, eot.AddDate(0, 0, 1))

	res.Today = dailyDetails{
		Yield:    remainingToday,
		Complete: !last.Before(eod),
	}
	res.Tomorrow = dailyDetails{
		Yield:    tomorrow,
		Complete: !last.Before(eot),
	}
	res.DayAfterTomorrow = dailyDetails{
		Yield:    dayAfterTomorrow,
		Complete: !last.Before(eot.AddDate(0, 0, 1)),
	}

	if err := site.collectors[metrics.Forecast].SetEnergy(forecastSlotEnergy(solar, time.Now())); err != nil {
		site.log.ERROR.Printf("solar forecast collector: %v", err)
	}

	if r, err := tariff.At(site.GetTariff(api.TariffUsageTemperature), time.Now()); err == nil {
		if err := site.collectors[metrics.Temperature].SetSocTemp(r.Value, true); err != nil {
			site.log.ERROR.Printf("temperature collector soc_temp: %v", err)
		}
	}

	res.Scale = site.solarScale()

	return res
}

// effectiveSolarScale returns the solar forecast scale used to adjust the
// optimizer's solar input if forecast adjustment is enabled, 1 otherwise.
func (site *Site) effectiveSolarScale() float64 {
	if !site.GetSolarAdjusted() {
		return 1
	}
	return site.solarScale()
}

const (
	solarScaleWindow     = 30  // trailing window of days to consider
	solarScaleMinSamples = 14  // minimum daily ratios before applying a scale
	solarScalePercentile = 0.5 // percentile of the daily ratio distribution to use
	solarScaleMinEnergy  = 0.5 // kWh, skip days where either side is too small for a meaningful ratio
)

// solarScale computes a scale factor for the solar forecast by sorting the daily
// produced/forecasted solar ratio over a trailing window of completed days and
// picking the value at a configured percentile (window: solarScaleWindow,
// percentile: solarScalePercentile). This captures the installation's systematic
// bias (soiling, shading, model error) instead of a single day's weather noise.
// The current (partial) day is excluded; returns 1 when there is not enough history.
//
// Depends only on completed days, so it's cached instead of recomputed per run.
func (site *Site) solarScale() float64 {
	scale, err := site.solarScaleCached()
	if err != nil {
		return 1
	}
	return scale
}

// querySolarScale does the actual metrics query and percentile calculation
// for solarScale, given the current beginning-of-day boundary.
func (site *Site) querySolarScale(bod time.Time) (float64, error) {
	from := bod.AddDate(0, 0, -solarScaleWindow)
	series, err := metrics.QueryEnergy(from, time.Now(), "day", true)
	if err != nil {
		return 0, err
	}

	pv := make(map[string]float64, solarScaleWindow)
	fcst := make(map[string]float64, solarScaleWindow)
	for _, s := range series {
		var m map[string]float64
		switch s.Group {
		case metrics.PV:
			m = pv
		case metrics.Forecast:
			m = fcst
		default:
			continue
		}
		for _, d := range s.Data {
			m[d.Start.Format("2006-01-02")] = d.Energy
		}
	}

	today := bod.Format("2006-01-02")
	ratios := make([]float64, 0, len(fcst))
	for day, f := range fcst {
		// skip today (partial) and dark days where the ratio is noise. The threshold
		// applies to production as well: a near-zero yield against a healthy forecast
		// is a fault (snow, soiling, inverter or metering outage), not a bias that
		// should be projected onto the next solarScaleWindow days.
		if p := pv[day]; day != today && f > solarScaleMinEnergy && p > solarScaleMinEnergy {
			ratios = append(ratios, p/f)
		}
	}

	scale, ok := percentileOf(ratios, solarScalePercentile, solarScaleMinSamples)
	if !ok {
		return 1, nil
	}
	site.log.DEBUG.Printf("solar scale P%.0f over %d days = %.3f", solarScalePercentile*100, len(ratios), scale)
	return scale, nil
}

// percentileOf returns the p-th percentile (0..1) of values by nearest-rank on the
// sorted series, or false when fewer than minSamples are present.
func percentileOf(values []float64, p float64, minSamples int) (float64, bool) {
	if len(values) < minSamples {
		return 0, false
	}
	s := slices.Clone(values)
	slices.Sort(s)
	return s[int(p*float64(len(s)-1))], true
}

func (site *Site) isDynamicTariff(usage api.TariffUsage) bool {
	tariff := site.GetTariff(usage)
	return tariff != nil && tariff.Type() != api.TariffTypePriceStatic
}
