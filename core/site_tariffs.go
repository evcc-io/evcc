package core

import (
	"encoding/json"
	"maps"
	"math"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/tariff"
	"github.com/samber/lo"
)

// greenShare returns
//   - the current green share, calculated for the part of the consumption between powerFrom and powerTo
//     the consumption below powerFrom will get the available green power first
func (site *Site) greenShare(powerFrom float64, powerTo float64) float64 {
	greenPower := math.Max(0, site.pvPower) + math.Max(0, site.batteryPower)
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

// accumulatedEnergy calculates the energy consumption between from and to,
// assuming the rates containing the power at given timestamp.
// Result is in Wh
func accumulatedEnergy(rr timeseries, from, to time.Time) float64 {
	var energy float64
	var last tsValue

	for _, r := range rr {
		// fmt.Println(r.Start.Local().Format(time.RFC3339), r.End.Local().Format(time.RFC3339), r.Price)

		if !r.Timestamp.After(from) {
			last = r
			continue
		}

		x1 := last.Timestamp
		y1 := last.Value
		if x1.Before(from) {
			x1 = from
			y1 += float64(from.Sub(last.Timestamp)) * (r.Value - last.Value) / float64(r.Timestamp.Sub(last.Timestamp))
		}

		x2 := r.Timestamp
		y2 := r.Value
		if x2.After(to) {
			x2 = to
			y2 += float64(to.Sub(r.Timestamp)) * (r.Value - last.Value) / float64(r.Timestamp.Sub(last.Timestamp))
		}

		energy += (y1 + y2) / 2 * x2.Sub(x1).Hours()

		if !r.Timestamp.Before(to) {
			break
		}

		last = r
	}

	return energy
}

type (
	timeseries []tsValue
	tsValue    struct {
		Timestamp time.Time `json:"ts"`
		Value     float64   `json:"val"`
	}
)

func (rr *timeseries) MarshalJSON() ([]byte, error) {
	return json.Marshal(rr)
}

func timestampSeries(rr api.Rates) timeseries {
	return lo.Map(rr, func(r api.Rate, _ int) tsValue {
		return tsValue{
			Timestamp: r.Start,
			Value:     r.Price,
		}
	})
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

	type dailyDetails struct {
		Yield    float64 `json:"energy"`
		Complete bool    `json:"complete"`
	}

	type solarDetails struct {
		Scale            *float64     `json:"scale,omitempty"`            // scale factor yield/forecasted today
		Today            dailyDetails `json:"today,omitempty"`            // tomorrow
		Tomorrow         dailyDetails `json:"tomorrow,omitempty"`         // tomorrow
		DayAfterTomorrow dailyDetails `json:"dayAfterTomorrow,omitempty"` // day after tomorrow
		Timeseries       timeseries   `json:"timeseries,omitempty"`       // timeseries of forecasted energy
	}

	fc := struct {
		Co2    api.Rates    `json:"co2,omitempty"`
		FeedIn api.Rates    `json:"feedin,omitempty"`
		Grid   api.Rates    `json:"grid,omitempty"`
		Solar  solarDetails `json:"solar,omitempty"`
	}{
		Co2:    tariff.Forecast(site.GetTariff(api.TariffUsageCo2)),
		FeedIn: tariff.Forecast(site.GetTariff(api.TariffUsageFeedIn)),
		Grid:   tariff.Forecast(site.GetTariff(api.TariffUsageGrid)),
	}

	// calculate adjusted solar forecast
	solar := timestampSeries(tariff.Forecast(site.GetTariff(api.TariffUsageSolar)))
	if len(solar) > 0 {
		fc.Solar.Timeseries = solar

		last := solar[len(solar)-1].Timestamp

		bod := beginningOfDay(time.Now())
		eod := bod.AddDate(0, 0, 1)
		eot := eod.AddDate(0, 0, 1)

		remainingToday := accumulatedEnergy(solar, time.Now(), eod)
		tomorrow := accumulatedEnergy(solar, eod, eot)
		dayAfterTomorrow := accumulatedEnergy(solar, eot, eot.AddDate(0, 0, 1))

		fc.Solar.Today = dailyDetails{
			Yield:    remainingToday,
			Complete: !last.Before(eod),
		}
		fc.Solar.Tomorrow = dailyDetails{
			Yield:    tomorrow,
			Complete: !last.Before(eot),
		}
		fc.Solar.DayAfterTomorrow = dailyDetails{
			Yield:    dayAfterTomorrow,
			Complete: !last.Before(eot.AddDate(0, 0, 1)),
		}

		// accumulate forecasted energy since last update
		site.fcstEnergy.AddEnergy(accumulatedEnergy(solar, site.fcstEnergy.updated, time.Now()) / 1e3)
		settings.SetFloat(keys.SolarAccForecast, site.fcstEnergy.Accumulated)

		produced := lo.SumBy(slices.Collect(maps.Values(site.pvEnergy)), func(v *meterEnergy) float64 {
			return v.AccumulatedEnergy()
		})

		if fcst := site.fcstEnergy.AccumulatedEnergy(); fcst > 0 {
			scale := produced / fcst
			site.log.DEBUG.Printf("solar forecast: accumulated %.1fkWh, produced %.1fkWh, scale %.1f", fcst, produced, scale)

			const minEnergy = 0.1
			if produced > minEnergy && fcst > minEnergy { /*kWh*/
				fc.Solar.Scale = lo.ToPtr(scale)
			}
		}
	}

	site.publish(keys.Forecast, fc)
}
