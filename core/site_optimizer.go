package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"sync/atomic"
	"time"

	evopt "github.com/andig/evopt/client"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
	"golang.org/x/exp/constraints"
)

var (
	eta          = float32(0.9)  // efficiency of the battery charging/discharging
	batteryPower = float32(6000) // default power of the battery in W

	updated time.Time
	mu      atomic.Uint32
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

const slotsPerHour = float64(time.Hour / tariff.SlotDuration)

func (site *Site) optimizerUpdateAsync(battery []measurement) {
	var err error

	if time.Since(updated) < 2*time.Minute {
		return
	}

	if !mu.CompareAndSwap(0, 1) {
		return
	}

	defer func() {
		updated = time.Now()
		mu.Store(0)

		if r := recover(); r != nil {
			err = fmt.Errorf("panic %v", r)
		}

		if err != nil {
			site.log.ERROR.Println("optimizer:", err)
		}
	}()

	err = site.optimizerUpdate(battery)
}

func (site *Site) optimizerUpdate(battery []measurement) error {
	uri := os.Getenv("EVOPT_URI")
	if uri == "" {
		return nil
	}

	solar := currentRates(site.GetTariff(api.TariffUsageSolar))
	grid := currentRates(site.GetTariff(api.TariffUsageGrid))
	feedIn := currentRates(site.GetTariff(api.TariffUsageFeedIn))

	minLen := lo.Min([]int{len(grid), len(feedIn), len(solar)})
	if minLen < 8 {
		return fmt.Errorf("not enough slots for optimization: %d (grid=%d, feedIn=%d, solar=%d)", minLen, len(grid), len(feedIn), len(solar))
	}

	dt := timeSteps(minLen)
	firstSlotDuration := time.Duration(dt[0]) * time.Second

	site.log.DEBUG.Printf("optimizer: optimizing %d slots until %v: grid=%d, feedIn=%d, solar=%d, first slot: %v",
		minLen,
		grid[minLen-1].End.Local(),
		len(grid), len(feedIn), len(solar),
		firstSlotDuration,
	)

	gt, err := site.homeProfile(minLen)
	if err != nil {
		return err
	}

	solarEnergy, err := solarRatesToEnergy(solar)
	if err != nil {
		return err
	}

	req := evopt.OptimizationInput{
		Strategy: evopt.OptimizerStrategy{
			ChargingStrategy:    evopt.OptimizerStrategyChargingStrategyChargeBeforeExport, // AttenuateGridPeaks
			DischargingStrategy: evopt.OptimizerStrategyDischargingStrategyDischargeBeforeImport,
		},
		EtaC: eta,
		EtaD: eta,
		TimeSeries: evopt.TimeSeries{
			Dt: dt,
			Gt: prorate(gt, firstSlotDuration),
			Ft: prorate(scaleAndPrune(solarEnergy, 1, minLen), firstSlotDuration),
			PN: scaleAndPrune(grid, 1e3, minLen),
			PE: scaleAndPrune(feedIn, 1e3, minLen),
		},
	}

	// end of horizon Wh value
	pa := lo.Min(req.TimeSeries.PN) * eta * 0.99

	details := responseDetails{
		Timestamps: asTimestamps(dt),
	}

	for _, lp := range site.Loadpoints() {
		// ignore disconnected loadpoints
		if lp.GetStatus() == api.StatusA {
			continue
		}

		v := lp.GetVehicle()
		if v == nil || v.Capacity() == 0 {
			continue
		}

		bat := evopt.BatteryConfig{
			ChargeFromGrid: true,
			CMin:           float32(lp.EffectiveMinPower()),
			CMax:           float32(lp.EffectiveMaxPower()),
			DMax:           0,
			SMin:           0,
			PA:             pa,
		}

		if profile := loadpointProfile(lp, minLen); profile != nil {
			bat.PDemand = prorate(profile, firstSlotDuration)
		}

		detail := batteryDetail{
			Type:  batteryTypeLoadpoint,
			Title: lp.GetTitle(),
		}

		// vehicle
		maxSoc := v.Capacity() * 1e3 // Wh
		if v := lp.EffectiveLimitSoc(); v > 0 {
			maxSoc *= float64(v) / 100
		} else if v := lp.GetLimitEnergy(); v > 0 {
			maxSoc = v * 1e3
		}

		bat.SInitial = float32(v.Capacity() * lp.GetSoc() * 10) // Wh
		bat.SMax = max(bat.SInitial, float32(maxSoc))           // prevent infeasible if current soc above maximum

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

		switch lp.GetMode() {
		case api.ModeOff:
			// disable charging
			bat.CMax = 0

		case api.ModeNow, api.ModeMinPV:
			// forced min/max charging
			if demand := continuousDemand(lp, minLen); demand != nil {
				bat.PDemand = prorate(demand, firstSlotDuration)
			}

		case api.ModePV:
			// add plan goal
			goal, socBased := lp.GetPlanGoal()
			if goal > 0 {
				if v := lp.GetVehicle(); socBased && v != nil {
					goal *= v.Capacity() * 10
				} else {
					goal *= 1000 // Wh
				}
			}

			if ts := lp.EffectivePlanTime(); !ts.IsZero() {
				// TODO precise slot placement
				if slot := int(time.Until(ts) / tariff.SlotDuration); slot < minLen {
					bat.SGoal = lo.RepeatBy(minLen, func(_ int) float32 { return 0 })
					bat.SGoal[slot] = float32(goal)
				} else {
					site.log.DEBUG.Printf("plan beyond forecast range: %.1f at %v", goal, ts.Round(time.Minute))
				}
			}

			// TODO remove once (using) smartcost limit becomes obsolete
			if costLimit := lp.GetSmartCostLimit(); costLimit != nil {
				maxLen := min(minLen, len(grid))

				// limit hit?
				if slices.ContainsFunc(grid[:maxLen], func(r api.Rate) bool {
					return r.Value <= *costLimit
				}) {
					maxPower := lp.EffectiveMaxPower()

					bat.PDemand = prorate(lo.RepeatBy(minLen, func(i int) float32 {
						return float32(maxPower / slotsPerHour)
					}), firstSlotDuration)

					for i := range maxLen {
						if grid[i].Value > *costLimit {
							bat.PDemand[i] = 0
						}
					}
				}
			}
		}

		req.Batteries = append(req.Batteries, bat)

		details.BatteryDetails = append(details.BatteryDetails, detail)
	}

	for i, b := range battery {
		if b.Capacity == nil || *b.Capacity == 0 || b.Soc == nil {
			continue
		}

		dev := site.batteryMeters[i]

		bat := evopt.BatteryConfig{
			CMax:     batteryPower,
			DMax:     batteryPower,
			SInitial: float32(*b.Capacity * *b.Soc * 10), // Wh
			PA:       pa,
		}
		bat.SMax = max(bat.SInitial, float32(*b.Capacity*1e3)) // Wh

		instance := dev.Instance()

		if _, ok := instance.(api.BatteryController); ok {
			bat.ChargeFromGrid = true
		}

		if m, ok := instance.(api.BatteryPowerLimiter); ok {
			charge, discharge := m.GetPowerLimits()
			bat.CMax = float32(charge)
			bat.DMax = float32(discharge)
		}

		if m, ok := instance.(api.BatterySocLimiter); ok {
			minSoc, maxSoc := m.GetSocLimits()
			bat.SMin = min(bat.SInitial, float32(*b.Capacity*minSoc*10)) // Wh
			bat.SMax = max(bat.SInitial, float32(*b.Capacity*maxSoc*10)) // Wh
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

	resp, err := apiClient.PostOptimizeChargeScheduleWithResponse(context.TODO(), req, func(_ context.Context, req *http.Request) error {
		if sponsor.IsAuthorized() {
			req.Header.Set("Authorization", "Bearer "+sponsor.Token)
		}
		return nil
	})
	if err != nil {
		return err
	}

	if resp.StatusCode() == http.StatusInternalServerError {
		return errors.New(resp.JSON500.Message)
	}

	if resp.StatusCode() == http.StatusBadRequest {
		return errors.New(resp.JSON400.Message)
	}

	if resp.StatusCode() != http.StatusOK {
		return fmt.Errorf("invalid status: %d", resp.StatusCode())
	}

	site.publish("evopt", struct {
		Req     evopt.OptimizationInput  `json:"req"`
		Res     evopt.OptimizationResult `json:"res"`
		Details responseDetails          `json:"details"`
	}{
		Req:     req,
		Res:     *resp.JSON200,
		Details: details,
	})

	return nil
}

// continuousDemand creates a slice of power demands depending on loadpoint mode
func continuousDemand(lp loadpoint.API, minLen int) []float32 {
	if lp.GetStatus() != api.StatusC {
		return nil
	}

	pwr := lp.EffectiveMaxPower()
	if lp.GetMode() == api.ModeMinPV {
		pwr = lp.EffectiveMinPower()
	}

	return lo.RepeatBy(minLen, func(i int) float32 {
		return float32(pwr / slotsPerHour)
	})
}

// loadpointProfile returns the loadpoint's charging profile in Wh
// TODO consider charging efficiency
func loadpointProfile(lp loadpoint.API, minLen int) []float64 {
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
	for range minLen {
		deltaEnergy := power * float64(tariff.SlotDuration) / float64(time.Hour) // Wh
		if energyKnown && deltaEnergy >= energy {
			deltaEnergy = energy
		}
		energy -= deltaEnergy

		res = append(res, deltaEnergy)
	}

	return res
}

// homeProfile returns the home base load in Wh
func (site *Site) homeProfile(minLen int) ([]float64, error) {
	// kWh over last 30 days
	profile, err := metrics.Profile(now.BeginningOfDay().AddDate(0, 0, -30))
	if err != nil {
		return nil, err
	}

	// max 4 days
	slots := make([]float64, 0, minLen+1)
	for len(slots) <= minLen+24*4 { // allow for prorating first day
		slots = append(slots, profile[:]...)
	}

	res := profileSlotsFromNow(slots)
	if len(res) < minLen {
		return nil, fmt.Errorf("minimum home profile length %d is less than required %d", len(res), minLen)
	}
	if len(res) > minLen {
		res = res[:minLen]
	}

	// convert to Wh
	return lo.Map(res, func(v float64, i int) float64 {
		return v * 1e3
	}), nil
}

// profileSlotsFromNow strips away any slots before "now".
// The profile contains 48 15min slots (00:00-23:45) that repeat for multiple days.
func profileSlotsFromNow(profile []float64) []float64 {
	firstSlot := int(time.Now().Truncate(tariff.SlotDuration).Sub(now.BeginningOfDay()) / tariff.SlotDuration)
	return profile[firstSlot:]
}

// prorate adjusts the first slot's energy amount according to remaining duration
func prorate[T constraints.Float](slots []T, firstSlotDuration time.Duration) []float32 {
	res := slices.Clone(slots)
	res[0] = res[0] * T(firstSlotDuration) / T(tariff.SlotDuration)
	return lo.Map(res, func(f T, _ int) float32 {
		return float32(f)
	})
}

func solarRatesToEnergy(rr api.Rates) (api.Rates, error) {
	res := make(api.Rates, 0, len(rr))

	for _, r := range rr {
		energy := solarEnergy(rr, r.Start, r.End)
		if energy < 0 {
			return nil, fmt.Errorf("negative solar energy from %v to %v: %.3f", r.Start, r.End, energy)
		}

		res = append(res, api.Rate{
			Start: r.Start,
			End:   r.End,
			Value: energy,
		})
	}

	return res, nil
}

func endOfHour(ts time.Time) time.Time {
	return ts.Truncate(time.Hour).Add(time.Hour)
}

func currentRates(tariff api.Tariff) api.Rates {
	if tariff == nil {
		return nil
	}

	rates, err := tariff.Rates()
	if err != nil {
		return nil
	}

	// filter past slots
	now := time.Now()
	return lo.Filter(rates, func(slot api.Rate, _ int) bool {
		return slot.End.After(now)
	})
}

func timeSteps(minLen int) []int {
	res := make([]int, 0, minLen)

	bos := time.Now().Truncate(tariff.SlotDuration)
	eos := bos.Add(tariff.SlotDuration)
	if d := time.Until(eos); d > time.Second && d < tariff.SlotDuration {
		res = append(res, int(d.Seconds()))
	}

	for i := len(res); i < minLen; i++ {
		res = append(res, int(tariff.SlotDuration.Seconds())) // 15min slots
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

func scaleAndPrune(rates api.Rates, div float64, maxLen int) []float32 {
	res := make([]float32, 0, maxLen)

	for _, slot := range rates {
		res = append(res, float32(slot.Value/div))
		if len(res) >= maxLen {
			break
		}
	}

	return res
}
