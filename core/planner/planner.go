package planner

import (
	"math"
	"slices"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	// DefaultMaxChargingWindows is the default for unlimited charging windows
	//DefaultMaxChargingWindows = 0
	
	// DefaultAllowSlotGap is the default to reduce costs
	DefaultAllowSlotGap = 1
)

// Planner plans a series of charging slots for a given (variable) tariff
type Planner struct {
	log                *util.Logger
	clock              clock.Clock // mockable time
	tariff             api.Tariff
	minGap             int // 0 = single window, 1 = cost-optimized, other = minimize charging windows (respect minimum window distance)
}

func New(log *util.Logger, tariff api.Tariff, opt ...func(t *Planner)) *Planner {
	p := &Planner{
		log:                log,
		clock:              clock.New(),
		tariff:             tariff,
		minGap:             DefaultAllowSlotGap,
	}

	for _, o := range opt {
		o(p)
	}

	return p
}

// chargingWindow represents a continuous charging period
type chargingWindow struct {
	totalCost float64
	start     time.Time
	end       time.Time
	duration  time.Duration
}

// calculateWindowCost calculates total cost for a time window
func (t *Planner) calculateWindowCost(rates api.Rates, start, end time.Time) float64 {
	var totalCost float64

	for _, rate := range rates {
		// skip slots outside our window
		if rate.End.Before(start) || rate.Start.After(end) {
			continue
		}

		// calculate overlap
		slotStart := rate.Start
		slotEnd := rate.End

		if slotStart.Before(start) {
			slotStart = start
		}
		if slotEnd.After(end) {
			slotEnd = end
		}

		duration := slotEnd.Sub(slotStart)
		totalCost += rate.Value * duration.Hours()
	}

	return totalCost
}

// findOptimalContinuousWindow finds the best continuous charging window
func (t *Planner) findOptimalContinuousWindow(rates api.Rates, requiredDuration time.Duration, targetTime time.Time) *chargingWindow {
	now := t.clock.Now()

	// earliest possible start
	earliestStart := now

	// latest possible start to finish before target
	latestStart := targetTime.Add(-requiredDuration)
	if latestStart.Before(now) {
		latestStart = now
	}

	var bestWindow *chargingWindow
	minCost := math.Inf(1)

	// create a time grid based on rate boundaries
	timePoints := make([]time.Time, 0)
	timePoints = append(timePoints, earliestStart)

	for _, rate := range rates {
		if rate.Start.After(earliestStart) && rate.Start.Before(targetTime) {
			timePoints = append(timePoints, rate.Start)
		}
		if rate.End.After(earliestStart) && rate.End.Before(targetTime) {
			timePoints = append(timePoints, rate.End)
		}
	}
	timePoints = append(timePoints, targetTime)

	// remove duplicates and sort
	slices.SortFunc(timePoints, func(a, b time.Time) int {
		return int(a.Sub(b))
	})
	timePoints = slices.Compact(timePoints)

	// try each possible start time
	for _, startTime := range timePoints {
		if startTime.Before(earliestStart) || startTime.After(latestStart) {
			continue
		}

		endTime := startTime.Add(requiredDuration)
		if endTime.After(targetTime) {
			continue
		}

		cost := t.calculateWindowCost(rates, startTime, endTime)

		if cost < minCost {
			minCost = cost
			bestWindow = &chargingWindow{
				start:     startTime,
				end:       endTime,
				duration:  requiredDuration,
				totalCost: cost,
			}
		}
	}

	return bestWindow
}

// buildPlanFromWindow creates a rate plan from a charging window
func (t *Planner) buildPlanFromWindow(rates api.Rates, window *chargingWindow) api.Rates {
	if window == nil {
		return nil
	}

	var plan api.Rates

	for _, rate := range rates {
		// skip slots outside window
		if rate.End.Before(window.start) || rate.Start.After(window.end) {
			continue
		}

		slot := rate

		// trim slot to window boundaries
		if slot.Start.Before(window.start) {
			slot.Start = window.start
		}
		if slot.End.After(window.end) {
			slot.End = window.end
		}

		plan = append(plan, slot)
	}

	// fill gaps if necessary
	plan = t.fillGaps(plan, window.start, window.end)

	return plan
}

// fillGaps fills any gaps in the plan with zero-cost slots
func (t *Planner) fillGaps(plan api.Rates, start, end time.Time) api.Rates {
	if len(plan) == 0 {
		return api.Rates{api.Rate{Start: start, End: end}}
	}

	plan.Sort()
	result := make(api.Rates, 0, len(plan)+2)

	// fill gap before first slot
	if plan[0].Start.After(start) {
		result = append(result, api.Rate{
			Start: start,
			End:   plan[0].Start,
		})
	}

	// add slots and fill gaps between them
	for i, slot := range plan {
		result = append(result, slot)

		if i < len(plan)-1 && slot.End.Before(plan[i+1].Start) {
			result = append(result, api.Rate{
				Start: slot.End,
				End:   plan[i+1].Start,
			})
		}
	}

	// fill gap after last slot
	if plan[len(plan)-1].End.Before(end) {
		result = append(result, api.Rate{
			Start: plan[len(plan)-1].End,
			End:   end,
		})
	}

	return result
}

// plan creates a lowest-cost plan or required duration (original logic)
func (t *Planner) plan(rates api.Rates, requiredDuration time.Duration, targetTime time.Time) api.Rates {
	var plan api.Rates

	for _, source := range rates {
		// slot not relevant
		if !(source.End.After(t.clock.Now()) && source.Start.Before(targetTime)) {
			continue
		}

		// adjust slot start and end
		slot := source
		if slot.Start.Before(t.clock.Now()) {
			slot.Start = t.clock.Now()
		}
		if slot.End.After(targetTime) {
			slot.End = targetTime
		}

		slotDuration := slot.End.Sub(slot.Start)
		requiredDuration -= slotDuration

		// slot covers more than we need, so shorten it
		if requiredDuration < 0 {
			// the first (if not single) slot should start as late as possible
			if IsFirst(slot, plan) && len(plan) > 0 {
				slot.Start = slot.Start.Add(-requiredDuration)
			} else {
				slot.End = slot.End.Add(requiredDuration)
			}
			requiredDuration = 0

			if slot.End.Before(slot.Start) {
				panic("slot end before start")
			}
		}

		plan = append(plan, slot)

		// we found all necessary slots
		if requiredDuration == 0 {
			break
		}
	}

	return plan
}

// continuousPlan creates a continuous emergency charging plan
func (t *Planner) continuousPlan(rates api.Rates, start, end time.Time) api.Rates {
	rates.Sort()

	res := make(api.Rates, 0, len(rates)+2)
	for _, r := range rates {
		// slot before continuous plan
		if !r.End.After(start) {
			continue
		}

		// slot after continuous plan
		if !r.Start.Before(end) {
			continue
		}

		// adjust first slot
		if r.Start.Before(start) && r.End.After(start) {
			r.Start = start
		}

		// adjust last slot
		if r.Start.Before(end) && r.End.After(end) {
			r.End = end
		}

		res = append(res, r)
	}

	if len(res) == 0 {
		res = append(res, api.Rate{
			Start: start,
			End:   end,
		})
	} else {
		// prepend missing slot
		if res[0].Start.After(start) {
			res = slices.Insert(res, 0, api.Rate{
				Start: start,
				End:   res[0].Start,
			})
		}
		// append missing slot
		if last := res[len(res)-1]; last.End.Before(end) {
			res = append(res, api.Rate{
				Start: last.End,
				End:   end,
			})
		}
	}

	return res
}

func (t *Planner) Plan(requiredDuration, precondition time.Duration, targetTime time.Time, minSlotGap ...int) api.Rates {
	if t == nil || requiredDuration <= 0 {
		return nil
	}

	t.minGap = DefaultAllowSlotGap
	if len(minSlotGap) > 0 {
		t.minGap = minSlotGap[0]
	}

	latestStart := targetTime.Add(-requiredDuration)
	if latestStart.Before(t.clock.Now()) {
		latestStart = t.clock.Now()
		targetTime = latestStart.Add(requiredDuration)
	}

	// simplePlan only considers time, but not cost
	simplePlan := api.Rates{
		api.Rate{
			Start: latestStart,
			End:   targetTime,
		},
	}

	// target charging without tariff or late start
	if t.tariff == nil {
		return simplePlan
	}

	rates, err := t.tariff.Rates()

	// treat like normal target charging if we don't have rates
	if len(rates) == 0 || err != nil {
		return simplePlan
	}

	// store original tariffs
	originalRates := slices.Clone(rates)

	// consume remaining time
	if t.clock.Until(targetTime) <= requiredDuration {
		return t.continuousPlan(rates, latestStart, targetTime)
	}

	// rates are by default sorted by date, oldest to newest
	last := rates[len(rates)-1].End

	// reduce planning horizon to available rates
	if targetTime.After(last) {
		// there is enough time for charging after end of current rates
		durationAfterRates := targetTime.Sub(last)
		if durationAfterRates >= requiredDuration {
			return nil
		}

		// need to use some of the available slots
		t.log.DEBUG.Printf("target time beyond available slots- reducing plan horizon from %v to %v",
			requiredDuration.Round(time.Second), durationAfterRates.Round(time.Second))

		targetTime = last
		requiredDuration -= durationAfterRates
	}

	// maxChargingWindows is 1, use continuous window optimization
	//if t.maxChargingWindows == 1 {
	if t.minGap == 0 {
		return t.planSingleWindow(rates, requiredDuration, precondition, targetTime)
	}

	// cost-optimized, sort rates by price and time
	slices.SortStableFunc(rates, sortByCost)

	// for late start ensure that the last slot is the cheapest
	var adjusted api.Rates
	if precondition > 0 {
		rates, adjusted = splitPreconditionSlots(rates, precondition, targetTime)
	}

	// sort rates by price and time
	slices.SortStableFunc(rates, sortByCost)

	plan := t.plan(rates, requiredDuration, targetTime)

	// correct plan slots to show original, non-adjusted prices
	for i, r := range plan {
		if rr, err := adjusted.At(r.Start); err == nil {
			plan[i].Value = rr.Value
		}
	}

	// minimize charging windows, avoid small gaps
	//if t.maxChargingWindows == 2 {
	if t.minGap > 1  {	
		plan = t.optimizeChargingWindows(plan, originalRates, requiredDuration)
		if precondition > 0 {
			plan = t.ensurePreconditioningWindow(plan, precondition, targetTime, originalRates)
		}
	}

	// recalculate costs
	plan = t.correctPricesFromOriginalRates(plan, originalRates)

	plan.Sort()

	return plan
}

// planSingleWindow creates an optimal single continuous charging window
func (t *Planner) planSingleWindow(rates api.Rates, requiredDuration, precondition time.Duration, targetTime time.Time) api.Rates {
	if len(rates) == 0 || requiredDuration <= 0 {
		return nil
	}

	// reduce required duration because of preconditioning
	effectiveDuration := requiredDuration
	if precondition > 0 && precondition <= requiredDuration {
		effectiveDuration -= precondition
	}

	var plan api.Rates

	if effectiveDuration > 0 {
		window := t.findOptimalContinuousWindow(rates, effectiveDuration, targetTime.Add(-precondition))
		if window == nil {
			t.log.WARN.Println("could not find optimal charging window")
			return nil
		}

		plan = t.buildPlanFromWindow(rates, window)
	}

	// preconditioning just before target time
	if precondition > 0 {
		preCondEnd := targetTime
		preCondStart := preCondEnd.Add(-precondition)

		for _, r := range rates {
			// within preconditioning window
			if r.End.After(preCondStart) && r.Start.Before(preCondEnd) {
				slot := r

				// trim slot to match preconditioning
				if slot.Start.Before(preCondStart) {
					slot.Start = preCondStart
				}
				if slot.End.After(preCondEnd) {
					slot.End = preCondEnd
				}

				plan = append(plan, slot)
			}
		}
	}

	plan.Sort()
	return plan
}

func splitPreconditionSlots(rates api.Rates, precondition time.Duration, targetTime time.Time) (api.Rates, api.Rates) {
	var res, adjusted api.Rates

	for _, r := range slices.Clone(rates) {
		preCondStart := targetTime.Add(-precondition)

		if !r.End.After(preCondStart) {
			res = append(res, r)
			continue
		}

		// split slot
		if !r.Start.After(preCondStart) {
			// keep the first part of the slot
			res = append(res, api.Rate{
				Start: r.Start,
				End:   preCondStart,
				Value: r.Value,
			})

			// adjust the second part of the slot
			r = api.Rate{
				Start: preCondStart,
				End:   r.End,
				Value: r.Value,
			}
		}

		// set the value to 0 to include slot in the plan
		res = append(res, api.Rate{
			Start: r.Start,
			End:   r.End,
			Value: 0,
		})

		// keep a copy of the adjusted slot
		adjusted = append(adjusted, r)
	}

	return res, adjusted
}

// avgPriceBetween calculates time-weighted average price from rates over interval [start, end)
func avgPriceBetween(rates api.Rates, start, end time.Time) float64 {
	if !end.After(start) {
		return 0
	}

	var totalCost float64
	var totalHours float64

	for _, r := range rates {
		// no overlap
		if r.End.Before(start) || r.Start.After(end) {
			continue
		}

		rs := r.Start
		re := r.End
		if rs.Before(start) {
			rs = start
		}
		if re.After(end) {
			re = end
		}

		d := re.Sub(rs).Hours()
		if d <= 0 {
			continue
		}

		totalCost += r.Value * d
		totalHours += d
	}

	if totalHours == 0 {
		return 0
	}
	return totalCost / totalHours
}

// mergeIntervalsWithAvg creates a new Rate entry [start..end) with time-weighted average value
func mergeIntervalsWithAvg(start, end time.Time, rates api.Rates) api.Rate {
	return api.Rate{
		Start: start,
		End:   end,
		Value: avgPriceBetween(rates, start, end),
	}
}

// bridgeSmallGaps connects plan slots if gap <= maxGapSlots * slotDuration
func (t *Planner) bridgeSmallGaps(plan api.Rates, fullRates api.Rates, maxGapSlots int) api.Rates {
	if len(plan) == 0 {
		return plan
	}
	if maxGapSlots <= 0 {
		return plan
	}

	// Determine slot duration
	var slotDuration time.Duration
	if len(fullRates) > 0 {
		slotDuration = fullRates[0].End.Sub(fullRates[0].Start)
	} else {
		slotDuration = plan[0].End.Sub(plan[0].Start)
	}
	if slotDuration <= 0 {
		return plan
	}

	maxGap := slotDuration * time.Duration(maxGapSlots)

	result := api.Rates{}
	current := plan[0]

	for i := 1; i < len(plan); i++ {
		next := plan[i]
		// if overlapping or continous -> extend current
		if !next.Start.After(current.End) {
			newEnd := current.End
			if next.End.After(newEnd) {
				newEnd = next.End
			}
			current = mergeIntervalsWithAvg(current.Start, newEnd, fullRates)
			continue
		}

		gap := next.Start.Sub(current.End)

		// if gap <= maxGap -> bridge
		if gap <= maxGap {
			merged := mergeIntervalsWithAvg(current.Start, next.End, fullRates)
			current = merged
			continue
		}

		// finish current and start next
		result = append(result, current)
		current = next
	}

	result = append(result, current)
	result.Sort()
	return result
}

// groupChargingWindows groups adjacent or overlapping slots
func (t *Planner) groupChargingWindows(plan api.Rates) api.Rates {
	if len(plan) == 0 {
		return plan
	}

	plan.Sort()
	grouped := api.Rates{}
	current := plan[0]

	for i := 1; i < len(plan); i++ {
		next := plan[i]

		// If next window touches or overlaps current → merge
		if !next.Start.After(current.End) || next.Start.Equal(current.End) {
			if next.End.After(current.End) {
				currDur := current.End.Sub(current.Start).Hours()
				nextDur := next.End.Sub(next.Start).Hours()
				totalDur := currDur + nextDur
				if totalDur > 0 {
					current.Value = (current.Value*currDur + next.Value*nextDur) / totalDur
				}
				current.End = next.End
			}
		} else {
			grouped = append(grouped, current)
			current = next
		}
	}
	grouped = append(grouped, current)
	grouped.Sort()
	return grouped
}

// trimToRequiredDurationWithRates trims based on real tariffs
func (t *Planner) trimToRequiredDurationWithRates(plan api.Rates, requiredDuration time.Duration, fullRates api.Rates) api.Rates {
	if len(plan) == 0 {
		return plan
	}

	total := time.Duration(0)
	for _, p := range plan {
		total += p.End.Sub(p.Start)
	}

	if total <= requiredDuration {
		return plan
	}

	excess := total - requiredDuration
	t.log.DEBUG.Printf("trimToRequiredDuration: need to remove %v of excess", excess)

	result := append(api.Rates{}, plan...)
	result.Sort()

	for excess > 0 && len(result) > 0 {
		first := result[0]
		last := result[len(result)-1]

		// Calculate real prices
		firstPrice := first.Value
		lastPrice := last.Value

		if fullRates != nil && len(fullRates) > 0 {
			firstPrice = avgPriceBetween(fullRates, first.Start, first.End)
			lastPrice = avgPriceBetween(fullRates, last.Start, last.End)
		}

		trimFromStart := false
		if firstPrice > lastPrice {
			trimFromStart = true
		}

		windowDur := last.End.Sub(last.Start)
		if trimFromStart {
			windowDur = first.End.Sub(first.Start)
		}

		if windowDur <= 0 {
			if trimFromStart {
				result = result[1:]
			} else {
				result = result[:len(result)-1]
			}
			continue
		}

		cut := minDuration(excess, windowDur)
		excess -= cut

		if trimFromStart {
			newStart := first.Start.Add(cut)
			if newStart.Before(first.End) {
				first.Start = newStart
				result[0] = first
			} else {
				result = result[1:]
			}
		} else {
			newEnd := last.End.Add(-cut)
			if newEnd.After(last.Start) {
				last.End = newEnd
				result[len(result)-1] = last
			} else {
				result = result[:len(result)-1]
			}
		}
	}

	result.Sort()
	return result
}

func minDuration(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// optimizeChargingWindows recursively optimizes to match window distance
func (t *Planner) optimizeChargingWindows(plan api.Rates, fullRates api.Rates, requiredDuration time.Duration) api.Rates {
	if len(plan) <= 1 {
		return plan
	}

	// sorted plan is pre-condition, ensure here
	plan.Sort()
	
	currentCost := t.calculateTotalCostWithRates(plan, fullRates)

	bridged := t.bridgeSmallGaps(plan, fullRates, t.minGap)
	grouped := t.groupChargingWindows(bridged)
	optimized := t.trimToRequiredDurationWithRates(grouped, requiredDuration, fullRates)

	newCost := t.calculateTotalCostWithRates(optimized, fullRates)

	t.log.DEBUG.Printf("optimizeChargingWindows: current cost €%.2f, new cost €%.2f, windows %d->%d",
		currentCost, newCost, len(plan), len(optimized))

	if newCost >= currentCost || len(optimized) == len(plan) {
		return optimized
	}

	return t.optimizeChargingWindows(optimized, fullRates, requiredDuration)
}

// calculateTotalCostWithRates based on real tariffs
func (t *Planner) calculateTotalCostWithRates(plan api.Rates, rates api.Rates) float64 {
	var totalCost float64
	for _, slot := range plan {
		avgPrice := avgPriceBetween(rates, slot.Start, slot.End)
		duration := slot.End.Sub(slot.Start).Hours()
		totalCost += avgPrice * duration
	}
	return totalCost
}

// correctPricesFromOriginalRates assigns correct prices to all slots
func (t *Planner) correctPricesFromOriginalRates(plan api.Rates, originalRates api.Rates) api.Rates {
	if len(plan) == 0 || len(originalRates) == 0 {
		return plan
	}

	corrected := make(api.Rates, len(plan))
	copy(corrected, plan)

	for i, slot := range corrected {
		avgPrice := avgPriceBetween(originalRates, slot.Start, slot.End)
		corrected[i].Value = avgPrice
	}

	return corrected
}

// ensurePreconditioningWindow adds missing preconditioning window
func (t *Planner) ensurePreconditioningWindow(plan api.Rates, precondition time.Duration, targetTime time.Time, originalRates api.Rates) api.Rates {
	if len(plan) == 0 || precondition <= 0 {
		return plan
	}

	preCondStart := targetTime.Add(-precondition)

	// Check if precond window exists
	for _, slot := range plan {
		if slot.Start.Before(targetTime) && slot.End.After(preCondStart) {
			return plan
		}
	}

	// Cut last slot by precondition duration
	// TODO cost optimization (pick best slot to cut)
	plan.Sort()
	plan[len(plan)-1].End = plan[len(plan)-1].End.Add(-precondition)

	// Add precond window
	for _, r := range originalRates {
		if r.End.After(preCondStart) && r.Start.Before(targetTime) {
			slot := r
			if slot.Start.Before(preCondStart) {
				slot.Start = preCondStart
			}
			if slot.End.After(targetTime) {
				slot.End = targetTime
			}
			plan = append(plan, slot)
		}
	}

	plan.Sort()
	return plan
}
