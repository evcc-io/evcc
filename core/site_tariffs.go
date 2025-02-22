package core

import (
	"math"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
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
func accumulatedEnergy(rr api.Rates, from, to time.Time) float64 {
	var energy float64
	var last api.Rate

	for _, r := range rr {
		// fmt.Println(r.Start.Local().Format(time.RFC3339), r.End.Local().Format(time.RFC3339), r.Price)

		if !r.Start.After(from) {
			last = r
			continue
		}

		start := last.Start
		if start.Before(from) {
			start = from
		}

		end := r.Start
		if end.After(to) {
			end = to
		}

		energy += (r.Price + last.Price) / 2 * end.Sub(start).Hours()

		if !r.Start.Before(to) {
			break
		}

		last = r
	}

	return energy
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

	// forecast

	solar := tariff.Forecast(site.GetTariff(api.TariffUsageSolar))

	type solarDetails struct {
		Forecast        api.Rates `json:"solar,omitempty"`
		ForecastedToday *float64  `json:"forecastedToday,omitempty"` // until now
		YieldToday      *float64  `json:"yieldToday,omitempty"`      // until now
	}

	fc := struct {
		Co2           api.Rates    `json:"co2,omitempty"`
		FeedIn        api.Rates    `json:"feedin,omitempty"`
		Grid          api.Rates    `json:"grid,omitempty"`
		Solar         api.Rates    `json:"solar,omitempty"`
		SolarAdjusted solarDetails `json:"adjusted,omitempty"`
	}{
		Co2:    tariff.Forecast(site.GetTariff(api.TariffUsageCo2)),
		FeedIn: tariff.Forecast(site.GetTariff(api.TariffUsageFeedIn)),
		Grid:   tariff.Forecast(site.GetTariff(api.TariffUsageGrid)),
		Solar:  solar,
	}

	// calculate adjusted solar forecast
	if solar != nil {
		forecastedToday := accumulatedEnergy(solar, beginningOfDay(time.Now()), time.Now())
		generatedToday := site.pvEnergy.AccumulatedEnergy()

		// TODO add lower limit for adjustment
		if forecastedToday > 0 && generatedToday > 0 {
			scale := generatedToday / forecastedToday

			var solarAdjusted api.Rates
			for _, r := range solar {
				solarAdjusted = append(solarAdjusted, api.Rate{
					Start: r.Start,
					End:   r.End,
					Price: r.Price * scale,
				})
			}

			fc.SolarAdjusted = solarDetails{
				Forecast:        solar,
				ForecastedToday: lo.ToPtr(forecastedToday),
				YieldToday:      lo.ToPtr(generatedToday),
			}
		}
	}

	site.publish(keys.Forecast, fc)
}
