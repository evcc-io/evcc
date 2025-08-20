package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	evopt "github.com/andig/evopt/client"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
	"moul.io/http2curl"
)

var (
	eta          = float32(0.9)  // efficiency of the battery charging/discharging
	batteryPower = float32(6000) // power of the battery in W

	updated time.Time
)

type batteryType string

const (
	batteryTypeLoadpoint batteryType = "loadpoint"
	batteryTypeVehicle   batteryType = "vehicle"
	batteryTypeBattery   batteryType = "battery"
)

type batteryDetail struct {
	Type     batteryType `json:"type"`
	Title    string      `json:"title,omitempty"`
	Name     string      `json:"name,omitempty"`
	Capacity float64     `json:"capacity,omitempty"`
}

type responseDetails struct {
	Timestamps     []time.Time     `json:"timestamp"`
	BatteryDetails []batteryDetail `json:"batteryDetails"`
}

func (site *Site) optimizerUpdateAsync(battery []measurement) {
	if time.Since(updated) < 5*time.Minute {
		return
	}

	var err error

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic %v", r)
		}

		if err != nil {
			site.log.ERROR.Println("optimizer:", err)
		}
	}()

	err = site.optimizerUpdate(battery)

	updated = time.Now()
}

func (site *Site) optimizerUpdate(battery []measurement) error {
	uri := os.Getenv("EVOPT_URI")
	if uri == "" {
		return nil
	}

	solarTariff := site.GetTariff(api.TariffUsageSolar)
	solarRates, err := solarTariff.Rates()
	if err != nil {
		return err
	}

	solar := currentSlots(solarTariff)
	grid := currentSlots(site.GetTariff(api.TariffUsageGrid))
	feedIn := currentSlots(site.GetTariff(api.TariffUsageFeedIn))

	minLen := lo.Min([]int{len(grid), len(feedIn), len(solar)})
	if minLen < 8 {
		return fmt.Errorf("not enough slots for optimization: %d", minLen)
	}

	dt := timeSteps(minLen)
	firstSlotDuration := time.Duration(dt[0]) * time.Second

	site.log.DEBUG.Printf("optimizer: optimizing %d slots until %v: grid=%d, feedIn=%d, solar=%d, first slot: %v",
		minLen,
		grid[minLen-1].End.Local(),
		len(grid), len(feedIn), len(solar),
		firstSlotDuration,
	)

	gt := site.homeProfile(minLen)

	solarEnergy, err := ratesToEnergy(solarRates, firstSlotDuration)
	if err != nil {
		return err
	}

	req := evopt.OptimizationInput{
		Strategy: &evopt.OptimizerStrategy{
			ChargingStrategy: lo.ToPtr(evopt.ChargeBeforeExport),
		},
		EtaC: &eta,
		EtaD: &eta,
		TimeSeries: evopt.TimeSeries{
			Dt: dt,
			Gt: asFloat32(gt),
			PN: maxValues(grid, 1e3, minLen),
			PE: maxValues(feedIn, 1e3, minLen),
			Ft: maxValues(solarEnergy, 1, minLen),
		},
	}

	// end of horizon Wh value
	pa := lo.Min(req.TimeSeries.PN)

	details := responseDetails{
		Timestamps: asTimestamps(dt),
	}

	for _, lp := range site.Loadpoints() {
		bat := evopt.BatteryConfig{
			ChargeFromGrid: lo.ToPtr(true),

			CMin: float32(lp.EffectiveMinPower()),
			CMax: float32(lp.EffectiveMaxPower()),
			DMax: 0,
			SMin: 0,
			PA:   pa,
		}

		if profile := loadpointProfile(lp, firstSlotDuration, minLen); profile != nil {
			bat.PDemand = lo.ToPtr(asFloat32(profile))
		}

		detail := batteryDetail{
			Type:  batteryTypeLoadpoint,
			Title: lp.GetTitle(),
		}

		if v := lp.GetVehicle(); v != nil {
			bat.SMax = float32(v.Capacity() * 1e3)                  // Wh
			bat.SInitial = float32(v.Capacity() * lp.GetSoc() * 10) // Wh

			detail.Type = batteryTypeVehicle
			detail.Capacity = v.Capacity()

			if vt := v.GetTitle(); vt != "" {
				if detail.Title != "" {
					detail.Title += " – "
				}
				detail.Title += vt
			}

			// find vehicle name/id
			for _, dev := range config.Vehicles().Devices() {
				if dev.Instance() == v {
					detail.Name = dev.Config().Name
				}
			}
		}

		req.Batteries = append(req.Batteries, bat)

		details.BatteryDetails = append(details.BatteryDetails, detail)
	}

	for i, b := range battery {
		if b.Capacity == nil || b.Soc == nil {
			continue
		}

		dev := site.batteryMeters[i]

		bat := evopt.BatteryConfig{
			CMin:     0,
			CMax:     batteryPower,
			DMax:     batteryPower,
			SMin:     0,
			SMax:     float32(*b.Capacity * 1e3),         // Wh
			SInitial: float32(*b.Capacity * *b.Soc * 10), // Wh
			PA:       pa,
		}

		// TODO atm we cannot cannot control charge from grid speed
		if _, ok := (dev.Instance()).(api.BatteryController); ok {
			bat.ChargeFromGrid = lo.ToPtr(true)
		}

		req.Batteries = append(req.Batteries, bat)

		details.BatteryDetails = append(details.BatteryDetails, batteryDetail{
			Type:     batteryTypeBattery,
			Name:     dev.Config().Name,
			Title:    deviceProperties(dev).Title,
			Capacity: *b.Capacity,
		})
	}

	httpClient := request.NewClient(site.log)
	httpClient.Timeout = 30 * time.Second

	apiClient, err := evopt.NewClientWithResponses(uri, evopt.WithHTTPClient(httpClient))
	if err != nil {
		return err
	}

	var curl *http2curl.CurlCommand
	resp, err := apiClient.PostOptimizeChargeScheduleWithResponse(context.TODO(), req, func(_ context.Context, req *http.Request) error {
		if sponsor.IsAuthorized() {
			req.Header.Set("Authorization", "Bearer "+sponsor.Token)
		}
		curl, _ = http2curl.GetCurlCommand(req)
		return nil
	})
	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusInternalServerError && resp.JSON500.Message != nil {
		return errors.New(*resp.JSON500.Message)
	}

	if resp.StatusCode() == http.StatusBadRequest && resp.JSON400.Message != nil {
		return errors.New(*resp.JSON400.Message)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("invalid status: %d", resp.StatusCode())
	}

	site.publish("evopt", struct {
		Req     evopt.OptimizationInput  `json:"req"`
		Res     evopt.OptimizationResult `json:"res"`
		Curl    string                   `json:"curl"`
		Details responseDetails          `json:"details"`
	}{
		Req:     req,
		Res:     *resp.JSON200,
		Curl:    curl.String(),
		Details: details,
	})

	return nil
}

// loadpointProfile returns the loadpoint's charging profile in Wh
// TODO consider charging efficiency
func loadpointProfile(lp loadpoint.API, firstSlotDuration time.Duration, minLen int) []float64 {
	mode := lp.GetMode()
	status := lp.GetStatus()

	if status != api.StatusC || (mode != api.ModeMinPV && mode != api.ModeNow) {
		return nil
	}

	power := lp.GetChargePower()
	if minP := lp.EffectiveMinPower(); mode == api.ModeMinPV && minP < power {
		power = minP
	}

	energy := lp.GetRemainingEnergy() * 1e3 // Wh
	energyKnown := energy > 0

	res := make([]float64, 0, minLen)
	for i := range minLen {
		d := 1.0 // hours
		if i == 0 {
			d = firstSlotDuration.Hours()
		}

		deltaEnergy := power * d // Wh
		if energyKnown && deltaEnergy >= energy {
			deltaEnergy = energy
		}
		energy -= deltaEnergy

		res = append(res, deltaEnergy)
	}

	return res
}

// homeProfile returns the home base load in Wh
func (site *Site) homeProfile(minLen int) []float64 {
	// kWh over last 30 days
	profile, err := metrics.Profile(now.BeginningOfDay().AddDate(0, 0, -30))
	if err != nil {
		site.log.WARN.Println("optimizer:", err)
		return lo.RepeatBy(minLen, func(_ int) float64 {
			return 0
		})
	}

	// max 4 days
	hours := make([]float64, 0, minLen+1)

	combined := combineSlots(profile[:])
	for len(hours) <= minLen+24 { // allow for prorating first day
		hours = append(hours, combined...)
	}

	res := prorateFirstHour(time.Now(), hours)
	if len(res) < minLen {
		panic("minimum home profile length failed")
	}
	if len(res) > minLen {
		res = res[:minLen]
	}

	// convert to Wh
	return lo.Map(res, func(v float64, i int) float64 {
		return v * 1e3
	})
}

// combineSlots combines 15-minute slots into hourly values
func combineSlots(profile []float64) []float64 {
	if profile == nil {
		return []float64{}
	}

	result := make([]float64, 0, 24)

	// Process complete hours starting from the start slot
	for hour := range 24 {
		var sum float64
		for i := range 4 {
			sum += profile[4*hour+i]
		}

		result = append(result, sum)
	}

	return result
}

// prorateFirstHour strips away any slots before "now" and prorates the first remaining hour
// based on remaining time in current hour. The profile contains hourly slots (0-23) that repeat for multiple days.
func prorateFirstHour(now time.Time, profile []float64) []float64 {
	// Take only slots from current hour onwards
	res := profile[now.Hour():]

	// Prorate the first hour based on remaining time in current hour
	if minutesIntoHour := now.Minute(); minutesIntoHour > 0 {
		fractionOfHour := float64(60-minutesIntoHour) / 60.0
		res[0] *= fractionOfHour
	}

	return res
}

func ratesToEnergy(rr api.Rates, firstSlot time.Duration) (api.Rates, error) {
	res := make(api.Rates, 0, len(rr))

	for _, r := range rr {
		from := r.Start

		if len(res) == 0 {
			from = endOfHour(r.Start).Add(-firstSlot)
		}

		if _, err := rr.At(from); err != nil {
			return nil, fmt.Errorf("missing solar data for: %v", from)
		}

		energy := solarEnergy(rr, from, r.End)
		if energy < 0 {
			return nil, fmt.Errorf("negative solar energy from %v to %v: %.3f", from, r.End, energy)
		}

		res = append(res, api.Rate{
			Start: from,
			End:   r.End,
			Value: energy,
		})
	}

	return res, nil
}

func asFloat32(gt []float64) []float32 {
	return lo.Map(gt, func(v float64, i int) float32 {
		return float32(v)
	})
}

func endOfHour(ts time.Time) time.Time {
	return ts.Truncate(time.Hour).Add(time.Hour)
}

func currentSlots(tariff api.Tariff) []api.Rate {
	if tariff == nil {
		return nil
	}

	rates, err := tariff.Rates()
	if err != nil {
		return nil
	}

	now := now.BeginningOfHour()
	return lo.Filter(rates, func(slot api.Rate, _ int) bool {
		return !slot.End.Before(now) // filter past slots
	})
}

func timeSteps(minLen int) []int {
	res := make([]int, 0, minLen)

	eoh := now.BeginningOfHour().Add(time.Hour)
	if d := time.Until(eoh); d > time.Second {
		res = append(res, int(d.Seconds()))
	}

	for i := len(res); i < minLen; i++ {
		res = append(res, 3600) // 1 hour in seconds
	}

	return res
}

func asTimestamps(dt []int) []time.Time {
	res := make([]time.Time, 0, len(dt))
	eoh := endOfHour(time.Now())
	res = append(res, eoh.Add(-time.Duration(dt[0])*time.Second))
	for i := range len(res) - 1 {
		res = append(res, eoh.Add(time.Duration(dt[i+1])*time.Second))
	}
	return res
}

func maxValues(rates []api.Rate, div float64, maxLen int) []float32 {
	res := make([]float32, 0, maxLen)

	for _, slot := range rates {
		res = append(res, float32(slot.Value/div))
		if len(res) >= maxLen {
			break
		}
	}

	return res
}
