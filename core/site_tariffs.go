package core

import (
	"maps"
	"math"
	"slices"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/tariff"
	"github.com/samber/lo"
)

type solarDetails struct {
	Scale            *float64     `json:"scale,omitempty"`            // scale factor yield/forecasted today
	Today            dailyDetails `json:"today,omitempty"`            // tomorrow
	Tomorrow         dailyDetails `json:"tomorrow,omitempty"`         // tomorrow
	DayAfterTomorrow dailyDetails `json:"dayAfterTomorrow,omitempty"` // day after tomorrow
	Timeseries       timeseries   `json:"timeseries,omitempty"`       // timeseries of forecasted energy
	Events           events       `json:"events,omitempty"`           // forecast-based events (experimental)
}

type dailyDetails struct {
	Yield    float64 `json:"energy"`
	Complete bool    `json:"complete"`
}

type events []event

type event struct {
	Timestamp  time.Time `json:"ts"`
	BatterySoc float64   `json:"batterySoc,omitempty"`
	Event      string    `json:"ev,omitempty"`
}

type loadpointStatus struct {
	Fixed           float64 `json:"fixed,omitempty"`
	Flexible        float64 `json:"flexible,omitempty"`
	RemainingEnergy float64 `json:"remainingEnergy,omitempty"`
}

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

	fc := struct {
		Co2    api.Rates     `json:"co2,omitempty"`
		FeedIn api.Rates     `json:"feedin,omitempty"`
		Grid   api.Rates     `json:"grid,omitempty"`
		Solar  *solarDetails `json:"solar,omitempty"`
	}{
		Co2:    tariff.Forecast(site.GetTariff(api.TariffUsageCo2)),
		FeedIn: tariff.Forecast(site.GetTariff(api.TariffUsageFeedIn)),
		Grid:   tariff.Forecast(site.GetTariff(api.TariffUsageGrid)),
	}

	// calculate adjusted solar forecast
	if solar := timestampSeries(tariff.Forecast(site.GetTariff(api.TariffUsageSolar))); len(solar) > 0 {
		fc.Solar = lo.ToPtr(site.solarDetails(solar))
	}

	site.publish(keys.Forecast, fc)
}

func (site *Site) solarDetails(solar timeseries) solarDetails {
	res := solarDetails{
		Timeseries: solar,
	}

	last := solar[len(solar)-1].Timestamp

	bod := beginningOfDay(time.Now())
	eod := bod.AddDate(0, 0, 1)
	eot := eod.AddDate(0, 0, 1)

	remainingToday := solar.energy(time.Now(), eod)
	tomorrow := solar.energy(eod, eot)
	dayAfterTomorrow := solar.energy(eot, eot.AddDate(0, 0, 1))

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

	// accumulate forecasted energy since last update
	site.fcstEnergy.AddEnergy(solar.energy(site.fcstEnergy.updated, time.Now()) / 1e3)
	settings.SetFloat(keys.SolarAccForecast, site.fcstEnergy.Accumulated)

	produced := lo.SumBy(slices.Collect(maps.Values(site.pvEnergy)), func(v *meterEnergy) float64 {
		return v.AccumulatedEnergy()
	})

	if fcst := site.fcstEnergy.AccumulatedEnergy(); fcst > 0 {
		scale := produced / fcst
		site.log.DEBUG.Printf("solar forecast: accumulated %.3fkWh, produced %.3fkWh, scale %.3f", fcst, produced, scale)

		const minEnergy = 0.5 // kWh
		if produced+fcst > minEnergy {
			res.Scale = lo.ToPtr(scale)
		}
	}

	if events := site.batteryForecast(solar); len(events) > 0 {
		res.Events = events
	}

	return res
}

const (
	batCharge    = "battery-charge"
	batDischarge = "battery-discharge"
	batSoc       = "battery-soc"
)

// batteryForecast projects the battery soc based on the solar forecast
func (site *Site) batteryForecast(solar timeseries) events {
	if site.batteryCapacity == 0 {
		return nil
	}

	const baseload = 300 // W

	forecastAvailable := solar.from(time.Now().Add(-time.Hour)).addConst(-baseload)
	if len(forecastAvailable) > 8 {
		forecastAvailable = forecastAvailable[:8]
	}

	// TODO check if all batteries have capacity and soc

	// defaults
	efficiency := soc.ChargeEfficiency
	minSoc := 20.0
	maxSoc := 95.0

	currentSoc := site.batterySoc
	prevSoc := site.batterySoc

	// initial entry
	prev := event{
		Timestamp:  time.Now(),
		Event:      batCharge,
		BatterySoc: site.batterySoc,
	}

	const slot = 15 * time.Minute

	ts := time.Now().Round(slot)
	if ts.Before(time.Now()) {
		prev.Timestamp = time.Now()
	}

	if delta := forecastAvailable.energy(prev.Timestamp, ts.Add(slot)); delta < 0 {
		prev.Event = batDischarge
	}

	res := events{prev}

	lps := make([]loadpointStatus, 0, len(site.loadpoints))
	for _, lp := range site.loadpoints {
		status := loadpointStatus{
			RemainingEnergy: lp.GetRemainingEnergy() / 1e3, // kWh
		}

		switch lp.GetMode() {
		case api.ModeNow:
			status.Fixed = lp.GetMaxPower()
		case api.ModeMinPV:
			status.Fixed = lp.GetMinPower()
		}

		lps = append(lps, status)
	}

	// create 15m slots
	for ts.Before(forecastAvailable[len(forecastAvailable)-1].Timestamp) {
		// fixed := lo.SumBy(lps, func(lp loadpointStatus) float64 {
		// 	if lp.RemainingEnergy > 0 {
		// 		return lp.Fixed
		// 	}
		// 	return 0
		// })
		// flexible := lo.SumBy(lps, func(lp loadpointStatus) float64 {
		// 	if lp.RemainingEnergy > 0 {
		// 		return lp.Flexible
		// 	}
		// 	return 0
		// })

		idx, _ := forecastAvailable.search(ts)
		fcst := forecastAvailable[idx]
		_ = fcst

		// forecastAvailable[idx].Value -= fixed
		// if forecastAvailable[idx].Value > 0 {
		// 	forecastAvailable[idx].Value -= min(forecastAvailable[idx].Value, flexible)
		// }

		for i, lp := range lps {
			if lp.RemainingEnergy <= 0 {
				continue
			}

			// fixed
			forecastAvailable[idx].Value -= lp.Fixed
			lps[i].RemainingEnergy = max(0, lps[i].RemainingEnergy-lp.Fixed*slot.Hours())

			// flexible
			if forecastAvailable[idx].Value > 0 {
				flexible := min(forecastAvailable[idx].Value, lp.Flexible)

				forecastAvailable[idx].Value -= flexible
				lps[i].RemainingEnergy = max(0, lps[i].RemainingEnergy-flexible*slot.Hours())
			}
		}

		// battery discharge control
		if forecastAvailable[idx].Value < 0 {
			forecastAvailable[idx].Value = 0
		}

		end := ts.Add(slot)
		energy := forecastAvailable.energy(ts, end)
		ts = end

		// add event
		ev := event{
			Timestamp: ts,
			Event:     batSoc,
		}

		switch {
		case energy > 0:
			energy *= efficiency
			if prev.Event != batCharge {
				ev.Event = batCharge
			}
		case energy < 0:
			energy /= efficiency
			if prev.Event != batDischarge {
				ev.Event = batDischarge
			}
		}

		currentSoc = min(max(currentSoc+energy/1e3/site.batteryCapacity*100, minSoc), maxSoc)
		ev.BatterySoc = math.Round(currentSoc)

		if ev.Event != batSoc || currentSoc != prevSoc {
			res = append(res, ev)
		}

		site.log.DEBUG.Printf("%+v %+v\n", ev, lps[0])

		// store previous soc
		prevSoc = currentSoc

		// store last charge/discharge
		if ev.Event != batSoc {
			prev = ev
		}
	}

	return res
}
