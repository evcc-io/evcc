package core

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/types"
	optimizer "github.com/evcc-io/optimizer/client"
	"github.com/samber/lo"
)

type batteryResult struct {
	batteryDetail
	Full  time.Time `json:"full,omitzero"`
	Empty time.Time `json:"empty,omitzero"`
}

// suggestionThreshold ignores numerical noise in power comparisons (W)
const suggestionThreshold = 50

// advisory actions for a loadpoint/vehicle slot; battery actions use api.BatteryMode
const (
	actionStop   = "stop"
	actionCharge = "charge"
)

// currentSlotSuggestion maps the optimizer's active-slot result onto an advisory action.
// Because the optimization is linear, the slot is at an operating-range extreme, so it
// maps cleanly onto the discrete battery mode / loadpoint intent that control would later apply.
// An idle battery is interpreted from the grid flow: importing means discharge is withheld
// (hold), exporting means charging is withheld (holdcharge). At its soc bound nothing is
// withheld, the bound does it, so the battery stays in normal mode. Without planned
// discharge the battery is held when holdWorthwhile, since normal mode would let the
// inverter cover intra-slot dips from the battery.
func currentSlotSuggestion(detail batteryDetail, req optimizer.BatteryConfig, res optimizer.BatteryResult, ts optimizer.TimeSeries, slot int, gridImport, gridExport float32, slotHours, pvShare float64) types.Suggestion {
	if slot < 0 || slotHours <= 0 || slot >= len(res.ChargingPower) || slot >= len(res.DischargingPower) {
		return types.Suggestion{}
	}

	charge := float64(res.ChargingPower[slot]) / slotHours
	discharge := float64(res.DischargingPower[slot]) / slotHours
	gridImporting := float64(gridImport)/slotHours > suggestionThreshold
	gridExporting := float64(gridExport)/slotHours > suggestionThreshold

	s := types.Suggestion{
		Charge:    charge,
		Discharge: discharge,
		Grid:      float64(gridImport-gridExport) / slotHours,
	}

	if detail.Type == batteryTypeBattery {
		idle := charge <= suggestionThreshold && discharge <= suggestionThreshold

		var empty, full bool
		if slot < len(res.StateOfCharge) {
			soc := res.StateOfCharge[slot]
			empty = soc <= req.SMin
			full = req.SMax > 0 && soc >= req.SMax
		}

		switch {
		case charge > pvShare+suggestionThreshold && gridImporting:
			// charging beyond its PV surplus share while importing means grid charging
			s.Action = api.BatteryCharge.String()
		case idle && gridImporting && !empty:
			// idle while importing: discharge is deliberately withheld
			s.Action = api.BatteryHold.String()
		case idle && gridExporting && !full:
			// idle while exporting: surplus is exported instead of charged
			s.Action = api.BatteryHoldCharge.String()
		case discharge > suggestionThreshold && gridExporting:
			// discharging while exporting means battery-to-grid discharge
			s.Action = api.BatteryDischarge.String()
		case discharge <= suggestionThreshold && !(idle && empty) && holdWorthwhile(req, res, ts, slot):
			s.Action = api.BatteryHold.String()
		default:
			s.Action = api.BatteryNormal.String()
		}

		// suggest the mode control applies instead of one the battery does not support
		if mode, err := api.BatteryModeString(s.Action); err == nil && detail.modes != nil {
			if mode = supportedBatteryMode(detail.modes, mode); mode != api.BatteryUnknown {
				s.Action = mode.String()
			}
		}
	} else if charge > suggestionThreshold {
		s.Action = actionCharge
	} else {
		s.Action = actionStop
	}

	return s
}

// holdWorthwhile reports whether energy kept in the battery now is discharged later at a
// higher price. A battery that fills up before its next discharge only displaces pv
// charging with the kept energy, so holding it would trade a battery dip for a grid import.
func holdWorthwhile(req optimizer.BatteryConfig, res optimizer.BatteryResult, ts optimizer.TimeSeries, slot int) bool {
	if slot >= len(ts.PN) {
		return false
	}

	for i := slot + 1; i < min(len(res.DischargingPower), len(ts.PN), len(ts.Dt)); i++ {
		if slotHours := float64(ts.Dt[i]) / 3600; slotHours > 0 && float64(res.DischargingPower[i])/slotHours > suggestionThreshold {
			return ts.PN[i] > ts.PN[slot]
		}
		if req.SMax > 0 && i < len(res.StateOfCharge) && res.StateOfCharge[i] >= req.SMax {
			return false
		}
	}

	return false
}

// optimizerPlan is a loadpoint's charging schedule from the last solve
type optimizerPlan struct {
	rates  api.Rates // charging slots valued at the grid import price
	energy []float64 // charge energy per slot, Wh
}

// loadpointPlan extracts the remaining charging slots of a battery result
func loadpointPlan(res optimizer.BatteryResult, prices []float32, schedule optimizerSchedule, now time.Time) optimizerPlan {
	var plan optimizerPlan

	for slot := range schedule.endsAfter(now) {
		if slot >= len(res.ChargingPower) || slot >= len(prices) {
			break
		}

		energy := float64(res.ChargingPower[slot])
		if energy/schedule.duration(slot).Hours() <= suggestionThreshold {
			continue
		}

		plan.energy = append(plan.energy, energy)
		plan.rates = append(plan.rates, api.Rate{
			Start: schedule.timestamps[slot],
			End:   schedule.end(slot),
			Value: float64(prices[slot]) * 1e3, // per Wh to per kWh
		})
	}

	return plan
}

// setBatteryForecast replaces the battery forecast of the cached state
func (site *Site) setBatteryForecast(forecast *types.BatteryForecast) {
	site.Lock()
	defer site.Unlock()

	site.battery.Forecast = forecast
}

// applyOptimizerResult maps the optimizer response onto suggestions, battery
// forecast and notifications
func (site *Site) applyOptimizerResult(solve optimizerSolve, now time.Time) {
	req, details, res, schedule := solve.req, solve.details, solve.res, solve.schedule

	slot := schedule.activeSlot(now)
	slotHours := schedule.duration(slot).Hours()
	var gridImport, gridExport float32
	if slot >= 0 && slot < len(res.GridImport) {
		gridImport = res.GridImport[slot]
	}
	if slot >= 0 && slot < len(res.GridExport) {
		gridExport = res.GridExport[slot]
	}

	// home battery discharge planned for the slot, a loadpoint charge it covers is no surplus
	var batteryDischarge, batteryCharge, pvSurplus float64
	if slot >= 0 && slotHours > 0 {
		for i, detail := range details.BatteryDetails {
			if detail.Type == batteryTypeBattery && slot < len(res.Batteries[i].DischargingPower) {
				batteryDischarge += float64(res.Batteries[i].DischargingPower[slot]) / slotHours
			}
			if detail.Type == batteryTypeBattery && slot < len(res.Batteries[i].ChargingPower) {
				batteryCharge += float64(res.Batteries[i].ChargingPower[slot]) / slotHours
			}
		}

		// PV surplus over home demand (gt, excluding loadpoints) covers home battery charging without grid charging (#35416).
		// Loadpoints are not deducted: the optimizer splits PV between them and the battery arbitrarily.
		if ts := req.TimeSeries; slot < len(ts.Ft) && slot < len(ts.Gt) {
			pvSurplus = max(0, float64(ts.Ft[slot]-ts.Gt[slot])/slotHours)
		}
	}

	var batteries []batteryResult
	suggestions := make(map[string]types.Suggestion, len(req.Batteries))

	for i, batReq := range req.Batteries {
		batRes := res.Batteries[i]
		detail := details.BatteryDetails[i]

		batteries = append(batteries, batteryResult{
			batteryDetail: detail,
			Full: matchSoc(batRes.StateOfCharge, schedule, now, func(soc float32) bool {
				return soc >= batReq.SMax
			}),
			Empty: matchSoc(batRes.StateOfCharge, schedule, now, func(soc float32) bool {
				return soc <= batReq.SMin
			}),
		})

		// home batteries share the PV surplus in proportion to their charge
		var pvShare float64
		if batteryCharge > 0 && slot >= 0 && slot < len(batRes.ChargingPower) {
			pvShare = pvSurplus * float64(batRes.ChargingPower[slot]) / slotHours / batteryCharge
		}

		suggestion := currentSlotSuggestion(detail, batReq, batRes, req.TimeSeries, slot, gridImport, gridExport, slotHours, pvShare)
		if suggestion.Action == "" {
			continue
		}

		if detail.loadpoint != nil {
			suggestion.BatteryDischarge = batteryDischarge
			suggestion.BatteryCharge = batteryCharge
		}

		// uncontrollable devices can't act on a suggestion
		if key := detail.key(); key != "" && detail.controllable {
			suggestions[key] = suggestion

			// the schedule is gated by the suggestion, so it needs no clearing.
			// Without the plan goal in the model its slots are no plan.
			if detail.loadpoint != nil {
				var plan optimizerPlan
				if batReq.SGoal != nil {
					plan = loadpointPlan(batRes, req.TimeSeries.PN, schedule, now)
				}
				site.loadpoints[*detail.loadpoint].setOptimizerPlan(plan)
			}
		}
	}

	site.publish("evopt-batteries", batteries)

	site.setSuggestions(suggestions)
	site.setBatteryForecast(site.addBatteryForecastTotals(req.Batteries, res.Batteries, schedule, now))

	site.publishBattery()

	// publish for all loadpoints so suggestions of dropped-out loadpoints clear
	site.publishSuggestions()

	// notify on actionable suggestion changes (advisory only, see #31903)
	for _, ev := range site.diffSuggestions(site.pendingSuggestions(details.BatteryDetails)) {
		site.pushEvent(ev)
	}

	solve.slot = slot
	site.setLastOptimizerSolve(&solve)
}

func (site *Site) addBatteryForecastTotals(req []optimizer.BatteryConfig, resp []optimizer.BatteryResult, schedule optimizerSchedule, now time.Time) *types.BatteryForecast {
	if len(resp) == 0 || len(resp[0].StateOfCharge) == 0 {
		return nil
	}

	high, low := batteryForecastSocExtremes(req, resp, schedule, now)
	if high == nil && low == nil {
		return nil
	}

	point := func(p *batteryForecastSlot) *types.BatteryForecastPoint {
		if p == nil {
			return nil
		}
		ts := schedule.end(p.slot)
		if !ts.After(now) {
			return nil
		}
		return &types.BatteryForecastPoint{Soc: p.soc, Time: ts, Limit: p.limit}
	}

	res := types.BatteryForecast{
		Highest: point(high),
		Lowest:  point(low),
	}
	if res.Highest == nil && res.Lowest == nil {
		return nil
	}
	return &res
}

type batteryForecastSlot struct {
	slot  int
	soc   float64 // percent
	limit bool    // true when SMax (highest) or SMin (lowest) boundary reached
}

// batteryForecastSocExtremes returns the highest and lowest aggregate SOC
// points across home batteries (SCapacity > 0) over the forecast horizon.
// The Limit flag indicates whether the SOC reached the configured SMax (for
// the highest point) or SMin (for the lowest point) boundary - in which case
// the battery is forecasted to become fully charged or empty.
// Returns nil for either point when no home battery is present or when the
// battery already is at the respective limit.
func batteryForecastSocExtremes(req []optimizer.BatteryConfig, resp []optimizer.BatteryResult, schedule optimizerSchedule, now time.Time) (*batteryForecastSlot, *batteryForecastSlot) {
	slot := schedule.activeSlot(now)
	homeIndices := lo.FilterMap(req, func(b optimizer.BatteryConfig, i int) (int, bool) {
		return i, b.SCapacity > 0
	})
	if len(homeIndices) == 0 || len(resp) == 0 || slot < 0 || slot >= len(resp[homeIndices[0]].StateOfCharge) {
		return nil, nil
	}

	totalCapacity := lo.SumBy(homeIndices, func(i int) float32 { return req[i].SCapacity })
	totalSMax := lo.SumBy(homeIndices, func(i int) float32 { return req[i].SMax })
	totalSMin := lo.SumBy(homeIndices, func(i int) float32 { return req[i].SMin })
	totalSInitial := lo.SumBy(homeIndices, func(i int) float32 {
		if slot > 0 {
			return resp[i].StateOfCharge[slot-1]
		}
		return req[i].SInitial
	})

	var high, low *batteryForecastSlot
	for i := range schedule.endsAfter(now) {
		if i >= len(resp[homeIndices[0]].StateOfCharge) {
			break
		}
		sum := lo.SumBy(homeIndices, func(idx int) float32 { return resp[idx].StateOfCharge[i] })
		soc := float64(sum/totalCapacity) * 100
		fullReached := totalSMax > 0 && sum >= totalSMax
		emptyReached := sum <= totalSMin

		// first slot at SMax wins for highest
		if high == nil || (!high.limit && (soc > high.soc || fullReached)) {
			high = &batteryForecastSlot{slot: i, soc: soc, limit: fullReached}
		}
		// first slot at SMin wins for lowest
		if low == nil || (!low.limit && (soc < low.soc || emptyReached)) {
			low = &batteryForecastSlot{slot: i, soc: soc, limit: emptyReached}
		}
	}

	// battery is already at the limit - announcing it will become full/empty is pointless
	if high != nil && high.limit && high.slot == slot && totalSInitial >= totalSMax {
		high = nil
	}
	if low != nil && low.limit && low.slot == slot && totalSInitial <= totalSMin {
		low = nil
	}

	return high, low
}

// matchSoc returns the end of the first slot whose soc satisfies fun.
func matchSoc(ts []float32, schedule optimizerSchedule, now time.Time, fun func(float32) bool) time.Time {
	for i := range schedule.endsAfter(now) {
		if i >= len(ts) {
			break
		}
		if fun(ts[i]) {
			return schedule.end(i)
		}
	}

	return time.Time{}
}
