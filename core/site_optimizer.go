package core

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strings"
	"sync/atomic"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	optimizer "github.com/evcc-io/optimizer/client"
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

type batteryResult struct {
	batteryDetail
	Full  *time.Time `json:"full"`
	Empty *time.Time `json:"empty"`
}

// func (br batteryResult) MarshalJSON() ([]byte, error) {
// 	var full, empty int64
// 	if !br.Full.IsZero() {
// 		full = int64(time.Until(br.Full).Seconds())
// 	}
// 	if !br.Empty.IsZero() {
// 		empty = int64(time.Until(br.Empty).Seconds())
// 	}

// 	return json.Marshal(struct {
// 		batteryResult
// 		UntilFull  int64 `json:"untilFull,omitempty"`
// 		UntilEmpty int64 `json:"untilEmpty,omitempty"`
// 	}{
// 		batteryResult: br,
// 		UntilFull:     full,
// 		UntilEmpty:    empty,
// 	})
// }

type requestDetails struct {
	Timestamps     []time.Time     `json:"timestamp"`
	BatteryDetails []batteryDetail `json:"batteryDetails"`
}

const slotsPerHour = float64(time.Hour / tariff.SlotDuration)

func (site *Site) optimizerUpdateAsync() {
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

	err = site.optimizerUpdate(site.battery.Devices)
}

func (site *Site) optimizerUpdate(battery []types.Measurement) error {
	uri := os.Getenv("OPTIMIZER_URI")
	if uri == "" {
		return nil
	}

	solarTariff := site.GetTariff(api.TariffUsageSolar)
	solar := currentRates(solarTariff)

	grid := currentRates(site.GetTariff(api.TariffUsageGrid))
	feedIn := currentRates(site.GetTariff(api.TariffUsageFeedIn))

	minLen := lo.Min([]int{len(grid), len(feedIn)})
	if solarTariff != nil {
		// allow empty solar forecast
		minLen = min(minLen, len(solar))
	}
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

	// allow empty solar forecast
	ft := lo.RepeatBy(minLen, func(i int) float32 { return float32(0) })
	if solarTariff != nil {
		solarEnergy, err := solarRatesToEnergy(solar)
		if err != nil {
			return err
		}

		ft = prorate(scaleAndPrune(solarEnergy, 1, minLen), firstSlotDuration)
	}

	req := optimizer.OptimizationInput{
		Strategy: optimizer.OptimizerStrategy{
			ChargingStrategy:    optimizer.OptimizerStrategyChargingStrategyChargeBeforeExport, // AttenuateGridPeaks
			DischargingStrategy: optimizer.OptimizerStrategyDischargingStrategyDischargeBeforeImport,
		},
		EtaC: eta,
		EtaD: eta,
		TimeSeries: optimizer.TimeSeries{
			Dt: dt,
			Gt: prorate(gt, firstSlotDuration),
			Ft: ft,
			PN: scaleAndPrune(grid, 1e3, minLen),
			PE: scaleAndPrune(feedIn, 1e3, minLen),
		},
	}

	// end of horizon Wh value
	pa := lo.Min(req.TimeSeries.PN) * eta * 0.99

	details := requestDetails{
		Timestamps: asTimestamps(dt),
	}

	if site.circuit != nil {
		if pMaxImp := site.circuit.GetMaxPower(); pMaxImp > 0 {
			req.Grid = optimizer.GridConfig{
				// hard grid import limit if no price penalty is set by PrcPExcImp
				PMaxImp: float32(pMaxImp),
			}
		}
	}

	add := func(battery optimizer.BatteryConfig, detail batteryDetail) {
		battery.PA = pa
		req.Batteries = append(req.Batteries, battery)
		details.BatteryDetails = append(details.BatteryDetails, detail)
	}

	for _, lp := range site.Loadpoints() {
		// ignore disconnected loadpoints
		if lp.GetStatus() == api.StatusA {
			continue
		}

		if v := lp.GetVehicle(); v == nil || v.Capacity() == 0 {
			continue
		}

		add(site.loadpointRequest(lp, minLen, firstSlotDuration, grid))
	}

	for i, dev := range site.batteryMeters {
		b := battery[i]

		if b.Capacity == nil || *b.Capacity == 0 || b.Soc == nil {
			continue
		}

		add(site.batteryRequest(dev, b, grid, minLen, firstSlotDuration))
	}

	httpClient := request.NewClient(site.log)
	httpClient.Timeout = 30 * time.Second

	apiClient, err := optimizer.NewClientWithResponses(uri, optimizer.WithHTTPClient(httpClient))
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

	if resp.StatusCode() != http.StatusOK {
		return apiError(resp)
	}

	if resp.JSON200.Status != optimizer.Optimal {
		return errors.New(string(resp.JSON200.Status))
	}

	site.publish("evopt", struct {
		Req     optimizer.OptimizationInput  `json:"req"`
		Res     optimizer.OptimizationResult `json:"res"`
		Details requestDetails               `json:"details"`
	}{
		Req:     req,
		Res:     *resp.JSON200,
		Details: details,
	})

	var batteries []batteryResult
	for i, batReq := range req.Batteries {
		batResp := resp.JSON200.Batteries[i]

		batResult := batteryResult{
			batteryDetail: details.BatteryDetails[i],
			Full: matchSoc(batResp.StateOfCharge, func(soc float32) bool {
				return soc >= batReq.SMax
			}),
			Empty: matchSoc(batResp.StateOfCharge, func(soc float32) bool {
				return soc <= batReq.SMin
			}),
		}

		batteries = append(batteries, batResult)
	}

	site.publish("evopt-batteries", batteries)

	site.battery.Forecast = site.addBatteryForecastTotals(req.Batteries, resp.JSON200.Batteries)

	site.publish(keys.Battery, site.battery)

	return nil
}

func (site *Site) addBatteryForecastTotals(req []optimizer.BatteryConfig, resp []optimizer.BatteryResult) *types.BatteryForecast {
	if len(resp) == 0 || len(resp[0].StateOfCharge) == 0 {
		return nil
	}

	now := time.Now().Round(tariff.SlotDuration)
	fullSlot, emptySlot := site.batteryForecastFullAndEmptySlots(req, resp)

	const zero = -1
	if fullSlot == zero && emptySlot == zero {
		return nil
	}

	var res types.BatteryForecast
	if fullSlot != zero {
		if ts := now.Add(time.Duration(fullSlot) * tariff.SlotDuration); ts.After(time.Now()) {
			res.Full = new(ts)
		}
	}
	if emptySlot != zero {
		if ts := now.Add(time.Duration(emptySlot) * tariff.SlotDuration); ts.After(time.Now()) {
			res.Empty = new(ts)
		}
	}

	return &res
}

func (site *Site) batteryForecastFullAndEmptySlots(req []optimizer.BatteryConfig, resp []optimizer.BatteryResult) (int, int) {
	matchSlot := func(fun func(soc float32, bat optimizer.BatteryConfig) bool) int {
	NEXT:
		for i := range resp[0].StateOfCharge {
			for batIdx := range req {
				if !fun(resp[batIdx].StateOfCharge[i], req[batIdx]) {
					continue NEXT
				}
			}
			return i
		}
		return -1
	}

	fullSlot := matchSlot(func(soc float32, bat optimizer.BatteryConfig) bool {
		return soc >= bat.SMax
	})
	emptySlot := matchSlot(func(soc float32, bat optimizer.BatteryConfig) bool {
		return soc <= bat.SMin
	})

	return fullSlot, emptySlot
}

func (site *Site) loadpointRequest(lp loadpoint.API, minLen int, firstSlotDuration time.Duration, grid api.Rates) (optimizer.BatteryConfig, batteryDetail) {
	bat := optimizer.BatteryConfig{
		ChargeFromGrid: true,
		CMin:           float32(lp.EffectiveMinPower()),
		CMax:           float32(lp.EffectiveMaxPower()),
		DMax:           0,
		SMin:           0,
		// PA:             pa,
	}

	if profile := loadpointProfile(lp, minLen); profile != nil {
		bat.PDemand = prorate(profile, firstSlotDuration)
	}

	detail := batteryDetail{
		Type:  batteryTypeLoadpoint,
		Title: lp.GetTitle(),
	}

	// vehicle
	v := lp.GetVehicle()

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

	var demand []float32

	switch lp.GetMode() {
	case api.ModeOff:
		// disable charging
		bat.CMax = 0

	case api.ModeNow:
		// forced max charging
		demand = continuousDemand(lp, minLen)

	case api.ModeMinPV:
		// forced min charging
		demand = continuousDemand(lp, minLen)
		// add smartcost limit and plan goal, if configured
		demand = applySmartCostLimit(lp, demand, grid, minLen)
		site.applyPlanGoal(lp, &bat, minLen)

	case api.ModePV:
		// add smartcost limit and plan goal, if configured
		demand = applySmartCostLimit(lp, nil, grid, minLen)
		site.applyPlanGoal(lp, &bat, minLen)
	}

	if demand != nil {
		bat.PDemand = prorate(demand, firstSlotDuration)
	}

	return bat, detail
}

func (site *Site) batteryRequest(dev config.Device[api.Meter], b types.Measurement, grid api.Rates, minLen int, firstSlotDuration time.Duration) (optimizer.BatteryConfig, batteryDetail) {
	bat := optimizer.BatteryConfig{
		CMax:      batteryPower,
		DMax:      batteryPower,
		SCapacity: float32(*b.Capacity * 1e3),         // Wh
		SInitial:  float32(*b.Capacity * *b.Soc * 10), // Wh
		// PA:       pa,
	}

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
		bat.SMin = float32(*b.Capacity * minSoc * 10) // Wh
		bat.SMax = float32(*b.Capacity * maxSoc * 10) // Wh
	}

	detail := batteryDetail{
		Type:     batteryTypeBattery,
		Name:     dev.Config().Name,
		Title:    deviceProperties(dev).Title,
		Capacity: *b.Capacity,
	}

	// tariff forecast-based grid charging demand
	if bat.ChargeFromGrid {
		if demand := site.applyBatteryGridChargeLimit(bat.CMax, grid, minLen); demand != nil {
			bat.PDemand = prorate(demand, firstSlotDuration)
		}
	}

	return bat, detail
}

func matchSoc(ts []float32, fun func(float32) bool) *time.Time {
	for i, soc := range ts {
		if fun(soc) {
			// TODO first slot
			return new(time.Now().Add(time.Duration(i+1) * tariff.SlotDuration))
		}
	}

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
	from := now.BeginningOfDay().AddDate(0, 0, -7)
	
	// kWh average over last 7 days - base load (excludes loadpoints)
	// Note: metrics.Profile() returns meter=1 which is calculated as:
	// gridPower + pvPower + batteryPower - totalChargePower
	// So it already excludes all loadpoint consumption
	gt_base, err := metrics.Profile(from)
	if err != nil {
		return nil, err
	}

	// Query heater profile (sum of all heating loadpoints)
	gt_heater_raw := site.extractHeaterProfile(from, time.Now())
	if gt_heater_raw != nil && len(gt_heater_raw) > 0 {
		site.log.DEBUG.Printf("home profile: extracted heater profile with %d slots", len(gt_heater_raw))
	}

	// max 4 days
	slots := make([]float64, 0, minLen+1)
	for len(slots) <= minLen+24*4 { // allow for prorating first day
		slots = append(slots, gt_base[:]...)
	}

	res := profileSlotsFromNow(slots)
	if len(res) < minLen {
		return nil, fmt.Errorf("minimum home profile length %d is less than required %d", len(res), minLen)
	}
	if len(res) > minLen {
		res = res[:minLen]
	}

	// If no heating devices or no heater data, return base load only
	if gt_heater_raw == nil || len(gt_heater_raw) == 0 {
		site.log.DEBUG.Println("home profile: no heating devices or heater data, returning base load only")
		// convert to Wh
		return lo.Map(res, func(v float64, i int) float64 {
			return v * 1e3
		}), nil
	}

	// Prepare heater profile with same length as base profile
	heaterSlots := make([]float64, 0, minLen+1)
	for len(heaterSlots) <= minLen+24*4 {
		heaterSlots = append(heaterSlots, gt_heater_raw[:]...)
	}
	gt_heater := profileSlotsFromNow(heaterSlots)
	if len(gt_heater) > len(res) {
		gt_heater = gt_heater[:len(res)]
	}

	// Try to apply temperature correction to heater profile
	// If correction cannot be applied (missing weather tariff, thresholds, etc.),
	// applyTemperatureCorrection returns the uncorrected profile
	site.log.DEBUG.Println("home profile: attempting temperature correction on heater profile")
	gt_heater_corrected := site.applyTemperatureCorrection(gt_heater)

	// Merge: final = base + heater (corrected or uncorrected)
	// Since base already excludes loadpoints, we always add the heating consumption
	// This ensures total household consumption includes heating even if correction fails
	gt_final := make([]float64, len(res))
	for i := range res {
		if i < len(gt_heater_corrected) {
			gt_final[i] = res[i] + gt_heater_corrected[i]
		} else {
			gt_final[i] = res[i]
		}
	}

	// convert to Wh
	return lo.Map(gt_final, func(v float64, i int) float64 {
		return v * 1e3
	}), nil
}

// applyTemperatureCorrection adjusts the household load profile based on outdoor temperature.
//
// The correction is gated on the 24h average actual temperature of the past 24 hours:
// if that average is at or above heatingThreshold, heating is considered off and no
// correction is applied to any slot.
//
// When heating is active (past 24h avg < threshold), for each future slot i:
//  1. Looks up the forecast temperature T_future at that slot's wall-clock time
//  2. Computes the average historical temperature T_past_avg at the same hour-of-day
//     from the past 7 days of Open-Meteo data already present in the rates slice
//  3. Applies: load[i] = load[i] * (1 + coeff * (T_past_avg - T_future))
//     A positive delta (tomorrow colder than historical average) increases the load estimate.
//     A negative delta (tomorrow warmer) decreases it.
func (site *Site) applyTemperatureCorrection(profile []float64) []float64 {
	weatherTariff := site.GetTariff(api.TariffUsageTemperature)
	if weatherTariff == nil {
		return profile
	}

	rates, err := weatherTariff.Rates()
	if err != nil || len(rates) == 0 {
		return profile
	}

	threshold := site.HeatingThreshold
	coeff := site.HeatingCoefficient
	
	// Require explicit configuration - no defaults
	// If either value is not configured (0), skip temperature correction
	if threshold == 0 || coeff == 0 {
		site.log.DEBUG.Println("temperature correction: heatingThreshold or heatingCoefficient not configured, skipping correction")
		return profile
	}

	currentTime := time.Now()

	// Compute the 24h average actual temperature from the past 24 hours.
	// Uses past rates (Start < now) within the last 24h window.
	yesterday := currentTime.Add(-24 * time.Hour)
	var pastSum24h float64
	var pastCount24h int
	for _, r := range rates {
		if !r.Start.Before(yesterday) && r.Start.Before(currentTime) {
			pastSum24h += r.Value
			pastCount24h++
		}
	}
	if pastCount24h == 0 {
		return profile
	}
	pastAvg24h := pastSum24h / float64(pastCount24h)

	// If the past 24h average actual temperature is at or above the heating threshold,
	// heating is considered off — no correction needed.
	if pastAvg24h >= threshold {
		return profile
	}

	// Pre-compute average historical temperature per hour-of-day (0..23) from past rates.
	// Past rates are those whose Start is before the current time.
	pastTempSum := make([]float64, 24)
	pastTempCount := make([]int, 24)
	for _, r := range rates {
		if r.Start.Before(currentTime) {
			h := r.Start.UTC().Hour()
			pastTempSum[h] += r.Value
			pastTempCount[h]++
		}
	}
	pastTempAvg := make([]float64, 24)
	for h := range 24 {
		if pastTempCount[h] > 0 {
			pastTempAvg[h] = pastTempSum[h] / float64(pastTempCount[h])
		}
	}

	result := make([]float64, len(profile))
	copy(result, profile)

	slotStart := currentTime.Truncate(tariff.SlotDuration)
	for i := range profile {
		ts := slotStart.Add(time.Duration(i) * tariff.SlotDuration)

		// Find the forecast temperature for this slot by direct timestamp match
		// Both weather data and profiles are on 15-minute slots, so direct matching works
		var tFuture float64
		found := false
		for _, r := range rates {
			if r.Start.Equal(ts) {
				tFuture = r.Value
				found = true
				break
			}
		}
		if !found {
			continue
		}

		h := ts.UTC().Hour()
		
		// Skip correction if no historical data for this hour
		if pastTempCount[h] == 0 {
			site.log.DEBUG.Printf("temperature correction: no historical data for hour %d, skipping slot %s", h, ts.Format("15:04"))
			continue
		}
		
		tPastAvg := pastTempAvg[h]

		// delta > 0: tomorrow colder than historical average → load increases
		// delta < 0: tomorrow warmer → load decreases
		delta := tPastAvg - tFuture
		result[i] = profile[i] * (1 + coeff*delta)
	}

	return result
}

// getHeatingLoadpoints returns the indices of all loadpoints with api.Heating feature
func (site *Site) getHeatingLoadpoints() []int {
	var heatingLPs []int
	for i, lp := range site.loadpoints {
		if hasFeature(lp.charger, api.Heating) {
			heatingLPs = append(heatingLPs, i)
		}
	}
	return heatingLPs
}

// extractHeaterProfile queries and aggregates consumption from all heating loadpoints
// Returns nil if no heating devices are configured
func (site *Site) extractHeaterProfile(from, to time.Time) []float64 {
	heatingLPs := site.getHeatingLoadpoints()
	if len(heatingLPs) == 0 {
		site.log.DEBUG.Println("heater profile: no heating loadpoints configured")
		return nil // no heating devices
	}

	site.log.DEBUG.Printf("heater profile: querying %d heating loadpoint(s)", len(heatingLPs))

	// Query each heating loadpoint's profile
	profiles := make([][]float64, 0, len(heatingLPs))
	for _, lpID := range heatingLPs {
		profile, err := metrics.LoadpointProfile(site.loadpoints[lpID].GetTitle(), from)
		if err == nil && profile != nil {
			site.log.DEBUG.Printf("heater profile: loadpoint %d has %d slots of data", lpID, len(profile))
			profiles = append(profiles, profile[:])
		} else {
			site.log.DEBUG.Printf("heater profile: loadpoint %d has no historical data", lpID)
		}
	}

	if len(profiles) == 0 {
		site.log.DEBUG.Println("heater profile: no historical data available from any heating loadpoint")
		return nil // no data available
	}

	// Sum profiles slot-by-slot
	result := sumProfiles(profiles)
	site.log.DEBUG.Printf("heater profile: aggregated %d heating loadpoint(s) into %d slots", len(profiles), len(result))
	return result
}

// sumProfiles sums multiple energy profiles slot-by-slot
func sumProfiles(profiles [][]float64) []float64 {
	if len(profiles) == 0 {
		return nil
	}

	// Use the length of the first profile as reference
	result := make([]float64, len(profiles[0]))
	for _, profile := range profiles {
		for i := 0; i < len(result) && i < len(profile); i++ {
			result[i] += profile[i]
		}
	}
	return result
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

func (site *Site) applyPlanGoal(lp loadpoint.API, bat *optimizer.BatteryConfig, minLen int) {
	goal, socBased := lp.GetPlanGoal()
	if goal <= 0 {
		return
	}

	// Convert to Wh
	if vehicle := lp.GetVehicle(); socBased && vehicle != nil {
		goal *= vehicle.Capacity() * 10
	} else {
		goal *= 1000 // Wh
	}

	ts := lp.EffectivePlanTime()
	if ts.IsZero() {
		return
	}

	// TODO precise slot placement
	slot := int(time.Until(ts) / tariff.SlotDuration)
	if slot >= 0 && slot < minLen {
		bat.SGoal = make([]float32, minLen)
		bat.SGoal[slot] = float32(goal)
		bat.SMax = max(bat.SMax, float32(goal))
	} else {
		site.log.DEBUG.Printf("plan beyond forecast range or overrun: %.1f at %v slot %d", goal, ts.Round(time.Minute), slot)
	}
}

// TODO remove once smart cost limit usage becomes obsolete
func applySmartCostLimit(lp loadpoint.API, demand []float32, grid api.Rates, minLen int) []float32 {
	costLimit := lp.GetSmartCostLimit()
	if costLimit == nil {
		return demand
	}

	maxLen := min(minLen, len(grid))

	// Check if any slots meet the cost limit
	if hasAffordableSlots := slices.ContainsFunc(grid[:maxLen], func(r api.Rate) bool {
		return r.Value <= *costLimit
	}); !hasAffordableSlots {
		return demand
	}

	maxPower := lp.EffectiveMaxPower()

	if demand == nil {
		demand = make([]float32, minLen)
	}

	for i := range maxLen {
		if grid[i].Value <= *costLimit {
			demand[i] = float32(maxPower / slotsPerHour)
		}
		// else: keep existing demand (either 0 or minPower from ModeMinPV)
	}

	return demand
}

func (site *Site) applyBatteryGridChargeLimit(cMax float32, grid api.Rates, minLen int) []float32 {
	limit := site.GetBatteryGridChargeLimit()
	if limit == nil {
		return nil
	}

	maxLen := min(minLen, len(grid))

	if hasAffordableSlots := slices.ContainsFunc(grid[:maxLen], func(r api.Rate) bool {
		return r.Value <= *limit
	}); !hasAffordableSlots {
		return nil
	}

	demand := make([]float32, minLen)
	for i := range maxLen {
		if grid[i].Value <= *limit {
			demand[i] = float32(float64(cMax) / slotsPerHour)
		}
	}

	return demand
}

// apiError extracts error message from optimizer API response
func apiError(resp *optimizer.PostOptimizeChargeScheduleResponse) error {
	var errObj *optimizer.Error
	switch resp.StatusCode() {
	case http.StatusBadRequest:
		errObj = resp.JSON400
	case http.StatusInternalServerError:
		errObj = resp.JSON500
	}

	if errObj == nil {
		return fmt.Errorf("invalid status: %d", resp.StatusCode())
	}

	if len(errObj.Details) > 0 {
		var details []string
		for field, msg := range errObj.Details {
			details = append(details, fmt.Sprintf("%s: %s", field, msg))
		}
		slices.Sort(details)
		return fmt.Errorf("%s (%s)", errObj.Message, strings.Join(details, ", "))
	}

	return errors.New(errObj.Message)
}
