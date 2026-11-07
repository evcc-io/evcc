package core

import (
	"errors"
	"fmt"
	"os"
	"slices"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/metrics"
	"github.com/evcc-io/evcc/core/types"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/grid"
	optimizer "github.com/evcc-io/optimizer/client"
	"github.com/jinzhu/now"
	"github.com/samber/lo"
	"golang.org/x/exp/constraints"
)

// optimizerDecaySlots is the number of slots over which measured values decay into the forecast
const optimizerDecaySlots = 4

type requestDetails struct {
	Timestamps     []time.Time     `json:"timestamp"`
	BatteryDetails []batteryDetail `json:"batteryDetails"`
	DemandDetails  []demandDetail  `json:"demandDetails"`
}

type demandType string

const (
	demandTypeHome       demandType = "home"
	demandTypeHeating    demandType = "heating"
	demandTypeUnmodelled demandType = "unmodelled"
)

// demandDetail is a single profile summarized into the home demand time series
type demandDetail struct {
	Type   demandType `json:"type"`
	Title  string     `json:"title,omitempty"`
	Values []float32  `json:"values"`
}

// endOfHorizonValue is the Wh value of energy left in storage at the end of the horizon, floored
// at pE/eta for storage that can export: charging surplus only stores eta Wh per Wh
func endOfHorizonValue(pN, pE []float32, eta float32, canExport bool) float32 {
	pa := lo.Min(pN) * eta * 0.99
	if canExport {
		pa = max(pa, lo.Min(pE)/eta*1.01)
	}
	return pa
}

// optimizerBattery pairs a battery request entry with its device detail
type optimizerBattery struct {
	cfg    optimizer.BatteryConfig
	detail batteryDetail
}

const slotsPerHour = float64(time.Hour / tariff.SlotDuration)

// optimizerRequest assembles the optimizer request and the matching device
// details from tariffs, home profile, loadpoints and battery meters
func (site *Site) optimizerRequest(battery []types.Measurement) (optimizer.OptimizationInput, requestDetails, error) {
	var req optimizer.OptimizationInput
	var details requestDetails

	solarTariff := site.GetTariff(api.TariffUsageSolar)
	solar := currentRates(solarTariff)

	// a configured but failing forecast is not a site without pv: a plan without sun
	// holds the battery for days. failing keeps the last plan until it expires.
	if solarTariff != nil && len(solar) == 0 {
		return req, details, errors.New("solar forecast unavailable")
	}

	grid := currentRates(site.GetTariff(api.TariffUsageGrid))
	planner := currentRates(site.GetTariff(api.TariffUsagePlanner))
	feedIn := feedInRates(site.GetTariff(api.TariffUsageFeedIn), grid)

	minLen := lo.Min([]int{len(grid), len(feedIn)})
	if solarTariff != nil {
		minLen = min(minLen, len(solar))
	}

	hosted := optimizerURI() == OPTIMIZER_URI
	if hosted {
		minLen = slotsUntil(grid, optimizerHorizon(time.Now()), minLen)
	}

	if expectedSlots := 8; minLen < expectedSlots {
		if solarTariff != nil {
			return req, details, fmt.Errorf("not enough forecast slots for meaningful optimization: %d < %d (grid=%d, feedIn=%d, solar=%d)", minLen, expectedSlots, len(grid), len(feedIn), len(solar))
		}
		return req, details, fmt.Errorf("not enough forecast slots for meaningful optimization: %d < %d (grid=%d, feedIn=%d)", minLen, expectedSlots, len(grid), len(feedIn))
	}

	now := time.Now()
	dt := timeSteps(minLen, now)
	firstSlotDuration := time.Duration(dt[0]) * time.Second

	site.log.DEBUG.Printf("optimizer: optimizing %d slots until %v: grid=%d, feedIn=%d, solar=%d, first slot: %v",
		minLen,
		grid[minLen-1].End.Local(),
		len(grid), len(feedIn), len(solar),
		firstSlotDuration,
	)

	gt, err := site.homeProfile(minLen)
	if err != nil {
		return req, details, err
	}

	// blend measured energy of the last metrics slot into the first slots
	if v := site.measuredSlotEnergy(metrics.Home); v > 0 {
		orig := slices.Clone(gt[:min(optimizerDecaySlots, len(gt))])
		blendMeasured(gt, v, optimizerDecaySlots)
		site.log.DEBUG.Printf("optimizer: home slots updated with measured %.0fWh: %.0f -> %.0f", v, orig, gt[:len(orig)])
	}

	// base load before the loadpoint contributions are summarized into gt
	home := slices.Clone(gt)

	// heating loadpoints add their forecast demand on top of the measured base load
	heaters := site.addHeatingDemand(gt, minLen)

	// sites without solar forecast plan without pv
	ft := lo.RepeatBy(minLen, func(i int) float32 { return float32(0) })
	if solarTariff != nil {
		solarEnergy, err := solarRatesToEnergy(solar)
		if err != nil {
			return req, details, err
		}

		ft = prorate(scaleAndPrune(solarEnergy, site.effectiveSolarScale(), minLen), firstSlotDuration)
	}

	req = optimizer.OptimizationInput{
		Strategy: optimizer.OptimizerStrategy{
			ChargingStrategy:    optimizer.OptimizerStrategyChargingStrategy(site.GetOptimizerChargingStrategy()),
			DischargingStrategy: optimizer.OptimizerStrategyDischargingStrategyDischargeBeforeImport,
		},
		EtaC: eta,
		EtaD: eta,
		TimeSeries: optimizer.TimeSeries{
			Dt: dt,
			Gt: prorate(gt, firstSlotDuration),
			Ft: ft,
			PN: scaleAndPrune(grid, 0.001, minLen),
			PE: scaleAndPrune(feedIn, 0.001, minLen),
		},
	}

	// individual profiles summarized into Gt, for the debug view
	details = requestDetails{
		Timestamps: asTimestamps(dt, now),
		DemandDetails: []demandDetail{
			{Type: demandTypeHome, Values: prorate(home, firstSlotDuration)},
		},
	}

	if site.circuit != nil {
		if pMaxImp := circuitMaxImport(site.circuit); pMaxImp > 0 {
			// hard grid import limit if no price penalty is set by PrcPExcImp
			req.Grid.PMaxImp = float32(pMaxImp)
		}
	}

	// static grid export limit configured in the UI: export is capped at this
	// power, excess PV is curtailed instead of exported
	if limit := site.GetGridExportLimit(); limit > 0 {
		req.Grid.PMaxExp = float32(limit)
	}

	// soft grid feed-in cap from active HEMS curtailment (e.g. German 70% rule)
	// wins over the static limit while active
	if curtailed := hems.Curtailed(site.hems); curtailed != nil && *curtailed {
		if pMaxExp := site.hems.MaxProductionPower(); pMaxExp != nil {
			req.Grid.PMaxExp = float32(*pMaxExp)
		}
	}

	var batteries []optimizerBattery

	for id, lp := range site.ActiveLoadpoints() {
		// ignore disconnected loadpoints, including StatusNone
		if s := lp.GetStatus(); s != api.StatusB && s != api.StatusC {
			continue
		}

		// heating loadpoints are already accounted for by their demand forecast
		if slices.ContainsFunc(heaters, func(h heatingDemand) bool { return h.lp == lp }) {
			continue
		}

		// no vehicle capacity and no session energy limit to model against:
		// account for the consumption as uncontrollable load
		if v := lp.GetVehicle(); v == nil || (v.Capacity() == 0 && lp.GetLimitEnergy() == 0) {
			site.addUnmodelledLoad(req.TimeSeries.Gt, &details, lp, minLen, firstSlotDuration)
			continue
		}

		// skip disabled loadpoints
		if cfg, detail := site.loadpointRequest(lp, minLen, firstSlotDuration, planner); cfg.CMax > 0 {
			detail.loadpoint = &id
			batteries = append(batteries, optimizerBattery{cfg, detail})
		}
	}

	// unmodelled loads precede the heating demand
	for _, h := range heaters {
		details.DemandDetails = append(details.DemandDetails, demandDetail{
			Type:   demandTypeHeating,
			Title:  h.lp.GetTitle(),
			Values: prorate(h.values, firstSlotDuration),
		})
	}

	for i, dev := range site.batteryMeters {
		// measurements may lag the configured meters on an off-cycle trigger
		if i >= len(battery) {
			break
		}
		b := battery[i]

		if b.Capacity == nil || *b.Capacity == 0 || b.Soc == nil {
			continue
		}

		cfg, detail := site.batteryRequest(dev, b, planner, minLen, firstSlotDuration)
		batteries = append(batteries, optimizerBattery{cfg, detail})
	}

	for _, b := range batteries {
		// vehicles cannot export
		b.cfg.PA = endOfHorizonValue(req.TimeSeries.PN, req.TimeSeries.PE, eta, b.detail.loadpoint == nil)
		req.Batteries = append(req.Batteries, b.cfg)
		details.BatteryDetails = append(details.BatteryDetails, b.detail)
	}

	// a lone base load profile is identical to the total
	if len(details.DemandDetails) < 2 {
		details.DemandDetails = nil
	}

	after, err := coarseSlotsAfter(hosted)
	if err != nil {
		site.log.WARN.Println("optimizer:", err)
	}
	if after > 0 {
		coarsenHorizon(&req, &details, after)
	}

	return req, details, nil
}

func (site *Site) loadpointRequest(lp loadpoint.API, minLen int, firstSlotDuration time.Duration, planner api.Rates) (optimizer.BatteryConfig, batteryDetail) {
	bat := optimizer.BatteryConfig{
		ChargeFromGrid: true,
		CActive:        lp.GetStatus() == api.StatusC,
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
		Type:         batteryTypeLoadpoint,
		Title:        lp.GetTitle(),
		controllable: true,
	}

	// vehicle
	v := lp.GetVehicle()

	capacity := v.Capacity() // kWh
	soc := lp.GetSoc()       // percent

	// without capacity or soc there is no battery state to model, but a session energy
	// limit still bounds the charge- use charged energy as state (see remainingLimitEnergy)
	if limit := lp.GetLimitEnergy(); limit > 0 && (capacity == 0 || soc == 0) {
		bat.SInitial = float32(lp.GetChargedEnergy())    // Wh
		bat.SMax = max(bat.SInitial, float32(limit*1e3)) // prevent infeasible if limit already exceeded
	} else {
		maxSoc := socToWh(capacity, float64(lp.EffectiveLimitSoc()))
		bat.SInitial = float32(socToWh(capacity, soc))
		if soc == 0 {
			// vehicle without soc: estimate stored energy from session charged energy
			bat.SInitial = float32(lp.GetChargedEnergy() * eta) // Wh
		}
		bat.SMax = max(bat.SInitial, float32(maxSoc)) // prevent infeasible if current soc above maximum
	}

	detail.Type = batteryTypeVehicle
	detail.Capacity = capacity

	if vt := v.GetTitle(); vt != "" {
		if detail.Title != "" {
			detail.Title += " (" + vt + ")"
		} else {
			detail.Title = vt
		}
	}

	// find vehicle name/id
	for _, dev := range config.Vehicles().Devices() {
		if dev.Instance() == v {
			detail.Name = dev.Config().Name
		}
	}

	var demand []float32
	automatic := site.AutomaticLoadpoints()

	switch lp.GetMode() {
	case api.ModeOff:
		// disable charging
		bat.CMax = 0

	case api.ModeNow:
		// forced max charging
		demand = continuousDemand(lp, minLen)

	case api.ModeSmart:
		if lp.GetAlwaysCharge().Active() {
			// forced min charging
			demand = continuousDemand(lp, minLen)
		}
		// add smartcost limit, precondition and plan goal, if configured
		demand = applySmartCostLimit(lp, demand, planner, minLen)
		if automatic {
			demand = applyPrecondition(lp, demand, minLen)
			site.applyPlanGoal(lp, &bat, minLen)
		}
	}

	// the loadpoint follows its own planner in any mode, which the optimizer must not reschedule
	if lp.GetMode() != api.ModeOff && !automatic {
		demand = applyPlannerPlan(lp, demand, bat.SInitial, minLen)
	}

	if demand != nil {
		// after prorate, so the shortened first slot counts with the energy it really carries
		bat.PDemand = clearDemandWhenFull(prorate(demand, firstSlotDuration), bat.SMax-bat.SInitial)
	}

	return bat, detail
}

// circuitMaxImport returns the circuit's import limit from max power incl. HEMS limit and three-phase max current, 0 if unlimited
func circuitMaxImport(c api.Circuit) float64 {
	res := c.EffectiveMaxPower()
	if current := c.GetMaxCurrent(); current > 0 {
		if power := grid.CurrentToPower(current, 3); res <= 0 || power < res {
			res = power
		}
	}
	return res
}

// clearDemandWhenFull zeroes the charge demand from the slot the accumulated energy fills the
// vehicle. The optimizer drops the demand at s_max anyway, but pays two binaries per slot to
// detect it, so slots that cannot bind are worth not asking about. Losses are accounted for.
//
// The cut assumes the demand is met every slot. A grid import limit can throttle charging below
// it, moving the real fill point later than the estimate - the next request corrects that from
// the measured soc, and the near slots are never affected because the cut sits a full charge away.
func clearDemandWhenFull(demand []float32, headroom float32) []float32 {
	res := slices.Clone(demand)

	var acc float32
	for i, d := range res {
		if acc >= headroom {
			res[i] = 0
			continue
		}
		acc += d * eta
	}

	return res
}

func (site *Site) batteryRequest(dev config.Device[api.Meter], b types.Measurement, planner api.Rates, minLen int, firstSlotDuration time.Duration) (optimizer.BatteryConfig, batteryDetail) {
	bat := optimizer.BatteryConfig{
		CMax:      batteryPower,
		DMax:      batteryPower,
		SCapacity: float32(*b.Capacity * 1e3), // Wh
		SInitial:  float32(socToWh(*b.Capacity, *b.Soc)),
		// PA:       pa,
	}
	bat.SMax = max(bat.SInitial, float32(*b.Capacity*1e3)) // Wh, narrowed by soc limits below

	instance := dev.Instance()

	ctrl, controllable := api.Cap[api.BatteryController](instance)
	if controllable {
		modes := ctrl.BatteryModes()
		bat.ChargeFromGrid = slices.Contains(modes, api.BatteryCharge)
		bat.DischargeToGrid = site.GetBatteryGridDischarge() && slices.Contains(modes, api.BatteryDischarge)
	}

	// a failed limit read keeps the defaults: a 0 limit would model a battery that never moves
	if m, ok := api.Cap[api.BatteryPowerLimiter](instance); ok {
		if charge, discharge, err := m.GetPowerLimits(); err == nil {
			bat.CMax = float32(charge)
			bat.DMax = float32(discharge)
		} else {
			site.log.WARN.Printf("optimizer: battery %s power limits: %v", deviceTitleOrName(dev), err)
		}
	}

	// soc limits measured by updateBatteryMeters; a failed read keeps the defaults
	if b.MinSoc != nil && b.MaxSoc != nil {
		minSoc, maxSoc := *b.MinSoc, *b.MaxSoc
		if maxSoc == 0 {
			maxSoc = 100 // empty/unset maxsoc means no upper limit
		}
		// clamp against current soc to prevent infeasible if it is outside the configured limits
		bat.SMin = min(bat.SInitial, float32(socToWh(*b.Capacity, minSoc)))
		bat.SMax = max(bat.SInitial, float32(socToWh(*b.Capacity, maxSoc)))
	}

	detail := batteryDetail{
		Type:         batteryTypeBattery,
		Name:         dev.Config().Name,
		Title:        deviceProperties(dev).Title,
		Capacity:     *b.Capacity,
		controllable: controllable,
	}
	if controllable {
		detail.modes = ctrl.BatteryModes()
	}

	// tariff forecast-based grid charging demand
	if bat.ChargeFromGrid {
		if demand := site.applyBatteryGridChargeLimit(bat.CMax, planner, minLen); demand != nil {
			bat.PDemand = prorate(demand, firstSlotDuration)
		}
	}

	// an external mode overrides evcc's battery control and is assumed to hold over the horizon
	if controllable {
		if mode := supportedBatteryMode(ctrl.BatteryModes(), site.GetBatteryModeExternal()); mode != api.BatteryUnknown {
			bat.ChargeFromGrid = mode == api.BatteryCharge
			bat.DischargeToGrid = mode == api.BatteryDischarge
			bat.PDemand = nil

			switch mode {
			case api.BatteryHold:
				bat.DMax = 0
			case api.BatteryHoldCharge, api.BatteryDischarge:
				bat.CMax = 0
			case api.BatteryCharge:
				bat.DMax = 0
				bat.PDemand = prorate(lo.RepeatBy(minLen, func(int) float32 {
					return float32(float64(bat.CMax) / slotsPerHour)
				}), firstSlotDuration)
			}
		}
	}

	return bat, detail
}

// continuousDemand creates a slice of power demands depending on loadpoint mode
func continuousDemand(lp loadpoint.API, minLen int) []float32 {
	if lp.GetStatus() != api.StatusC {
		return nil
	}

	pwr := lp.EffectiveMaxPower()
	if loadpoint.AlwaysChargeActive(lp) {
		pwr = lp.EffectiveMinPower()
	}

	return lo.RepeatBy(minLen, func(i int) float32 {
		return float32(pwr / slotsPerHour)
	})
}

// loadpointProfile returns the loadpoint's charging profile in Wh
// TODO consider charging efficiency
func loadpointProfile(lp loadpoint.API, minLen int) []float64 {
	minActive := loadpoint.AlwaysChargeActive(lp)

	if lp.GetStatus() != api.StatusC || (!minActive && lp.GetMode() != api.ModeNow) {
		return nil
	}

	power := lp.GetChargePower()
	if minP := lp.EffectiveMinPower(); minActive && minP < power {
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

// addUnmodelledLoad adds the uncontrollable power of a loadpoint to the home demand,
// which the home profile does not contain
func (site *Site) addUnmodelledLoad(gt []float32, details *requestDetails, lp loadpoint.API, minLen int, firstSlotDuration time.Duration) {
	power := unmodelledPower(lp)
	if power <= 0 {
		return
	}

	// no forecast and no capacity to assert it further, so the measured power only decays
	load := make([]float64, minLen)
	blendMeasured(load, power/slotsPerHour, optimizerDecaySlots)

	site.log.DEBUG.Printf("optimizer: home slots updated with unmodelled %.0fW load of %s: %.0f", power, lp.GetTitle(), load[:min(optimizerDecaySlots, len(load))])

	prorated := prorate(load, firstSlotDuration)
	for i, v := range prorated {
		gt[i] += v
	}

	details.DemandDetails = append(details.DemandDetails, demandDetail{
		Type:   demandTypeUnmodelled,
		Title:  lp.GetTitle(),
		Values: prorated,
	})
}

// unmodelledPower returns the uncontrollable power of a connected loadpoint that
// cannot be modelled as storage because the vehicle capacity is unknown
func unmodelledPower(lp loadpoint.API) float64 {
	power := lp.GetChargePower()

	// always charge keeps drawing at least min power while the vehicle is connected,
	// even before the charge meter has caught up
	if loadpoint.AlwaysChargeActive(lp) && lp.GetStatus() == api.StatusC {
		power = max(power, lp.EffectiveMinPower())
	}

	return max(0, power)
}

// measuredSlotEnergy returns the energy in Wh of the last completed
// metrics slot for the given collector ref, 0 when not available
func (site *Site) measuredSlotEnergy(ref string) float64 {
	c, ok := site.collectors[ref]
	if !ok {
		return 0
	}

	v, _ := c.LastSlotEnergy()
	return v * 1e3
}

// blendMeasured decays the first slots from the measured value into the
// forecast. Slot 0 uses the measured value, the forecast takes over from
// slot decaySlots on.
func blendMeasured[T constraints.Float](slots []T, measured T, decaySlots int) {
	for i := range min(decaySlots, len(slots)) {
		w := T(decaySlots-i) / T(decaySlots)
		slots[i] = w*measured + (1-w)*slots[i]
	}
}

// prorate adjusts the first slot's energy amount according to remaining duration
func prorate[T constraints.Float](slots []T, firstSlotDuration time.Duration) []float32 {
	// return empty slice instead of nil to make api happy
	if len(slots) == 0 {
		return []float32{}
	}

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

// feedInRates returns the feed-in rates, or zero-priced grid slots without feed-in tariff
func feedInRates(tariff api.Tariff, grid api.Rates) api.Rates {
	if tariff != nil {
		return currentRates(tariff)
	}

	return lo.Map(grid, func(slot api.Rate, _ int) api.Rate {
		return api.Rate{Start: slot.Start, End: slot.End}
	})
}

// optimizerHorizon is the timeframe the hosted optimizer is limited to for sake
// of performance: 48 hours, extended to the end of that day. In the early hours
// the extension would add almost a full day, hence it only applies past 6:00.
func optimizerHorizon(t time.Time) time.Time {
	horizon := t.Add(48 * time.Hour)
	if t.Hour() < 6 {
		return horizon
	}
	return now.With(horizon).EndOfDay()
}

// slotsUntil limits maxLen to the slots starting before the given horizon
func slotsUntil(rates api.Rates, horizon time.Time, maxLen int) int {
	if i := slices.IndexFunc(rates[:min(maxLen, len(rates))], func(slot api.Rate) bool {
		return slot.Start.After(horizon)
	}); i >= 0 {
		return i
	}
	return maxLen
}

func timeSteps(minLen int, now time.Time) []int {
	res := make([]int, 0, minLen)

	eos := now.Truncate(tariff.SlotDuration).Add(tariff.SlotDuration)
	if d := eos.Sub(now); d > time.Second && d < tariff.SlotDuration {
		res = append(res, int(d.Seconds()))
	}

	for i := len(res); i < minLen; i++ {
		res = append(res, int(tariff.SlotDuration.Seconds())) // 15min slots
	}

	return res
}

func asTimestamps(dt []int, now time.Time) []time.Time {
	res := make([]time.Time, 0, len(dt))

	eos := now.Truncate(tariff.SlotDuration).Add(tariff.SlotDuration)
	res = append(res, eos.Add(-time.Duration(dt[0])*time.Second))

	for i := range len(dt) - 1 {
		res = append(res, res[i].Add(time.Duration(dt[i])*time.Second))
	}

	return res
}

// defaultCoarseSlotsAfter is where the request switches from quarter hour to clock hour slots: the long
// horizons are the requests that take the optimizer's slow split path and set the hosted p99.
const defaultCoarseSlotsAfter = 48 * time.Hour

// coarseSlotsAfter returns where the request switches to clock hour slots. A self-hosted optimizer
// may move it with OPTIMIZER_COARSEN_AFTER in hours, 0 keeps quarter hour slots for the full horizon.
func coarseSlotsAfter(hosted bool) (time.Duration, error) {
	v := os.Getenv("OPTIMIZER_COARSEN_AFTER")
	if hosted || v == "" {
		return defaultCoarseSlotsAfter, nil
	}

	h, err := strconv.Atoi(v)
	if err != nil || h < 0 {
		return defaultCoarseSlotsAfter, fmt.Errorf("invalid OPTIMIZER_COARSEN_AFTER %q, using %v", v, defaultCoarseSlotsAfter)
	}

	return time.Duration(h) * time.Hour, nil
}

// coarsenHorizon merges the quarter hour slots beyond after into clock hours. Quarter slots bridge
// to the first full hour, and a plan goal ends its slot so the departure keeps its time. Energies
// are summed, prices averaged by duration.
func coarsenHorizon(req *optimizer.OptimizationInput, details *requestDetails, after time.Duration) {
	dt, ts := req.TimeSeries.Dt, details.Timestamps

	goalEnds := func(i int) bool {
		return slices.ContainsFunc(req.Batteries, func(b optimizer.BatteryConfig) bool {
			return i < len(b.SGoal) && b.SGoal[i] != 0
		})
	}

	// first slot index of every output slot; hourly latches at the first full hour past the boundary
	starts := make([]int, 0, len(dt))
	coarse := ts[0].Add(after)
	var hourly bool
	for i := range dt {
		if hourly && ts[i].Minute() != 0 && !goalEnds(i-1) {
			continue
		}
		starts = append(starts, i)
		hourly = hourly || (!ts[i].Before(coarse) && ts[i].Minute() == 0)
	}
	if len(starts) == len(dt) {
		return
	}
	ends := append(slices.Clone(starts[1:]), len(dt))

	coarseDt := lo.Map(starts, func(s, g int) int { return lo.Sum(dt[s:ends[g]]) })
	sum := func(v []float32) []float32 {
		if v == nil {
			return nil
		}
		return lo.Map(starts, func(s, g int) float32 { return lo.Sum(v[s:ends[g]]) })
	}
	mean := func(v []float32) []float32 {
		return lo.Map(starts, func(s, g int) float32 {
			var weighted float32
			for i := s; i < ends[g]; i++ {
				weighted += v[i] * float32(dt[i])
			}
			return weighted / float32(coarseDt[g])
		})
	}

	req.TimeSeries.Dt = coarseDt
	req.TimeSeries.Gt = sum(req.TimeSeries.Gt)
	req.TimeSeries.Ft = sum(req.TimeSeries.Ft)
	req.TimeSeries.PN = mean(req.TimeSeries.PN)
	req.TimeSeries.PE = mean(req.TimeSeries.PE)
	for i := range req.Batteries {
		b := &req.Batteries[i]
		b.PDemand = sum(b.PDemand)
		if b.SGoal != nil {
			// a goal ends its group, so it sits in the group's last slot
			b.SGoal = lo.Map(starts, func(s, g int) float32 { return b.SGoal[ends[g]-1] })
		}
	}
	for i := range details.DemandDetails {
		details.DemandDetails[i].Values = sum(details.DemandDetails[i].Values)
	}
	details.Timestamps = lo.Map(starts, func(s, _ int) time.Time { return ts[s] })
}

func scaleAndPrune(rates api.Rates, scale float64, maxLen int) []float32 {
	res := make([]float32, 0, maxLen)

	for _, slot := range rates {
		res = append(res, float32(slot.Value*scale))
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

	vehicle := lp.GetVehicle()
	socBased = socBased && vehicle != nil
	goal = planGoalWh(lp, goal, socBased, bat.SInitial)

	ts := lp.EffectivePlanTime()
	if ts.IsZero() {
		return
	}

	now := time.Now()

	slot := planSlot(now, ts)
	if slot >= 0 && slot < minLen {
		setPlanGoal(bat, minLen, slot, goal)
	} else {
		site.log.DEBUG.Printf("plan beyond forecast range or overrun: %.1f at %v slot %d", goal, ts.Round(time.Minute), slot)
	}

	// later plans let the optimizer charge ahead for them instead of replanning once the first goal is reached.
	// Capped at s_max: the loadpoint only charges beyond its limit for the effective plan.
	if socBased {
		for _, g := range lp.GetSocPlanGoals() {
			if slot := planSlot(now, g.Time); slot >= 0 && slot < minLen {
				setPlanGoal(bat, minLen, slot, min(socToWh(vehicle.Capacity(), float64(g.Soc)), float64(bat.SMax)))
			}
		}
	}
}

// planGoalWh converts the plan goal to Wh. Energy plans count from the energy
// charged since the plan was set, on top of the current state.
func planGoalWh(lp loadpoint.API, goal float64, socBased bool, sInitial float32) float64 {
	if v := lp.GetVehicle(); socBased && v != nil {
		return socToWh(v.Capacity(), goal)
	}
	return float64(sInitial) + lp.GetPlanRemainingEnergy()*1e3
}

// setPlanGoal sets the goal (Wh) at slot, keeping a higher goal already set there
func setPlanGoal(bat *optimizer.BatteryConfig, minLen, slot int, goal float64) {
	if bat.SGoal == nil {
		bat.SGoal = make([]float32, minLen)
	}
	bat.SGoal[slot] = max(bat.SGoal[slot], float32(goal))
	bat.SMax = max(bat.SMax, float32(goal))
}

// socToWh converts a soc in % of a capacity in kWh to Wh
func socToWh(capacity, soc float64) float64 {
	return capacity * soc * 10
}

// planSlot returns the first slot whose end is at or after ts, since s_goal[i] models the SoC at the end of slot i.
// Slot i ends at eos+i*SlotDuration on the timeSteps timeline; -1 if ts is not in the future.
func planSlot(now, ts time.Time) int {
	if !ts.After(now) {
		return -1
	}

	eos := now.Truncate(tariff.SlotDuration).Add(tariff.SlotDuration)
	return int((ts.Sub(eos) + tariff.SlotDuration - 1) / tariff.SlotDuration)
}

// applyPlannerPlan adds the planner's charging schedule as fixed demand, including precondition and continuous strategy
func applyPlannerPlan(lp loadpoint.API, demand []float32, sInitial float32, minLen int) []float32 {
	ts := lp.EffectivePlanTime()
	if ts.IsZero() {
		return demand
	}

	goal, socBased := lp.GetPlanGoal()
	energy := planGoalWh(lp, goal, socBased, sInitial) - float64(sInitial)
	if energy <= 0 {
		return demand
	}

	maxPower := lp.EffectiveMaxPower()
	required := lp.GetPlanRequiredDuration(goal, maxPower)
	if required <= 0 {
		return demand
	}

	strategy := lp.EffectivePlanStrategy()
	plan := lp.GetPlan(ts, required, strategy.Precondition, strategy.Continuous)

	return planDemand(plan, demand, energy/eta, time.Now(), minLen)
}

// planDemand spreads the plan's grid energy (Wh) evenly over its slots, which also covers the planner's
// extra time for tapering. Slot 0 is the remainder of the current slot, its energy relative to the full
// slot like the other demands before prorate.
func planDemand(plan api.Rates, demand []float32, energy float64, now time.Time, minLen int) []float32 {
	total := lo.SumBy(plan, func(r api.Rate) time.Duration { return r.End.Sub(r.Start) })
	if total <= 0 {
		return demand
	}

	if demand == nil {
		demand = make([]float32, minLen)
	}

	eos := now.Truncate(tariff.SlotDuration).Add(tariff.SlotDuration)

	for i := range minLen {
		start, end := eos.Add(time.Duration(i-1)*tariff.SlotDuration), eos.Add(time.Duration(i)*tariff.SlotDuration)
		if i == 0 {
			start = now
		}

		var overlap time.Duration
		for _, r := range plan {
			from, to := r.Start, r.End
			if from.Before(start) {
				from = start
			}
			if to.After(end) {
				to = end
			}
			if from.Before(to) {
				overlap += to.Sub(from)
			}
		}

		if overlap > 0 {
			slotEnergy := energy * float64(overlap) / float64(total)
			demand[i] = max(demand[i], float32(slotEnergy*float64(tariff.SlotDuration)/float64(end.Sub(start))))
		}
	}

	return demand
}

// TODO remove once smart cost limit usage becomes obsolete
func applySmartCostLimit(lp loadpoint.API, demand []float32, planner api.Rates, minLen int) []float32 {
	costLimit := lp.GetSmartCostLimit()
	if costLimit == nil {
		return demand
	}

	maxLen := min(minLen, len(planner))

	// Check if any slots meet the cost limit
	if hasAffordableSlots := slices.ContainsFunc(planner[:maxLen], func(r api.Rate) bool {
		return r.Value <= *costLimit
	}); !hasAffordableSlots {
		return demand
	}

	maxPower := lp.EffectiveMaxPower()

	if demand == nil {
		demand = make([]float32, minLen)
	}

	for i := range maxLen {
		if planner[i].Value <= *costLimit {
			demand[i] = float32(maxPower / slotsPerHour)
		}
		// else: keep existing demand (either 0 or minPower from always charge)
	}

	return demand
}

// applyPrecondition forces max charging power during the planner's precondition window
// ("late charging"), i.e. the last precondition duration before the plan time
func applyPrecondition(lp loadpoint.API, demand []float32, minLen int) []float32 {
	precondition := lp.EffectivePlanStrategy().Precondition
	if precondition <= 0 {
		return demand
	}

	ts := lp.EffectivePlanTime()
	if ts.IsZero() {
		return demand
	}

	// limit to the required charging duration, i.e. "all" must not demand beyond the plan goal
	goal, _ := lp.GetPlanGoal()
	if required := lp.GetPlanRequiredDuration(goal, lp.EffectiveMaxPower()); required < precondition {
		precondition = required
	}
	if precondition <= 0 {
		return demand
	}

	// TODO precise slot placement
	end := time.Until(ts)
	start := end - precondition
	if end <= 0 {
		return demand
	}

	first := max(int(start/tariff.SlotDuration), 0)
	if first >= minLen {
		return demand
	}

	if demand == nil {
		demand = make([]float32, minLen)
	}

	energy := float32(lp.EffectiveMaxPower() / slotsPerHour)

	for i := first; i < minLen; i++ {
		slotStart := time.Duration(i) * tariff.SlotDuration
		overlap := min(end, slotStart+tariff.SlotDuration) - max(start, slotStart)
		if overlap <= 0 {
			break
		}

		demand[i] = max(demand[i], energy*float32(overlap)/float32(tariff.SlotDuration))
	}

	return demand
}

func (site *Site) applyBatteryGridChargeLimit(cMax float32, planner api.Rates, minLen int) []float32 {
	limit := site.GetBatteryGridChargeLimit()
	if limit == nil {
		return nil
	}

	maxLen := min(minLen, len(planner))

	if hasAffordableSlots := slices.ContainsFunc(planner[:maxLen], func(r api.Rate) bool {
		return r.Value <= *limit
	}); !hasAffordableSlots {
		return nil
	}

	demand := make([]float32, minLen)
	for i := range maxLen {
		if planner[i].Value <= *limit {
			demand[i] = float32(float64(cMax) / slotsPerHour)
		}
	}

	return demand
}
