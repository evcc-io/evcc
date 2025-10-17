package planner

import (
	"math"
	"slices"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

// Planner plans a series of charging slots for a given (variable) tariff
type Planner struct {
	log    *util.Logger
	clock  clock.Clock // mockable time
	tariff api.Tariff
}

// New creates a price planner
func New(log *util.Logger, tariff api.Tariff, opt ...func(t *Planner)) *Planner {
	p := &Planner{
		log:    log,
		clock:  clock.New(),
		tariff: tariff,
	}

	for _, o := range opt {
		o(p)
	}

	return p
}

// plan creates a lowest-cost plan or required duration.
// It MUST already established that
// - rates are sorted in ascending order by cost and descending order by start time (prefer late slots)
// - target time and required duration are before end of rates
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

// Plan creates a continuous emergency charging plan
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

	// if no slots remain, create a full slot
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

// findOptimalContinuousWindow finds the cheapest continuous time window of the given duration
// within the available rates, ending no later than targetTime.
//
// The algorithm:
// 1. Generates all possible window start points from rate boundaries
// 2. For each valid window, calculates the total cost based on overlapping rate slots
// 3. Returns the window with minimal cost as a single merged slot with weighted average price
//
// Returns: best plan as a single rate slot with weighted average price, and the total cost
func (t *Planner) findOptimalContinuousWindow(rates api.Rates, effectiveDuration time.Duration, targetTime time.Time) (api.Rates, float64) {
	now := t.clock.Now()

	// Validate inputs
	if len(rates) == 0 || effectiveDuration <= 0 {
		t.log.TRACE.Printf("findOptimalContinuousWindow: no rates or invalid duration (rates=%d, duration=%v)", len(rates), effectiveDuration)
		return nil, 0
	}

	rates.Sort() // sort slots by start time

	// Collect all relevant time points (rate boundaries + now + target)
	points := make([]time.Time, 0, 2*len(rates)+2)
	for _, r := range rates {
		points = append(points, r.Start, r.End)
	}
	points = append(points, now, targetTime)

	// Sort and remove duplicates
	slices.SortFunc(points, func(a, b time.Time) int {
		return a.Compare(b)
	})
	points = slices.Compact(points)

	t.log.TRACE.Printf("findOptimalContinuousWindow: now=%v, targetTime=%v, effectiveDuration=%v, points=%d",
		now.Round(time.Second), targetTime.Round(time.Second), effectiveDuration, len(points))

	var bestPlan api.Rates
	minCost := math.Inf(1)
	windowsChecked := 0

	// Try each possible window start position
	for _, windowStart := range points {
		// Skip windows starting in the past
		if windowStart.Before(now) {
			continue
		}

		windowEnd := windowStart.Add(effectiveDuration)

		// Window must end at or before target time
		if windowEnd.After(targetTime) {
			break
		}

		windowsChecked++

		// Calculate cost for this window by examining overlapping rates
		totalCost := 0.0
		windowPlan := make(api.Rates, 0)

		for _, rate := range rates {
			// Skip rates that don't overlap with this window
			if rate.End.Before(windowStart) || rate.Start.After(windowEnd) {
				continue
			}

			// Calculate the overlapping portion
			overlapStart := rate.Start
			if windowStart.After(overlapStart) {
				overlapStart = windowStart
			}
			overlapEnd := rate.End
			if windowEnd.Before(overlapEnd) {
				overlapEnd = windowEnd
			}

			if overlapStart.Before(overlapEnd) {
				duration := overlapEnd.Sub(overlapStart).Hours()
				totalCost += rate.Value * duration

				windowPlan = append(windowPlan, api.Rate{
					Start: overlapStart,
					End:   overlapEnd,
					Value: rate.Value,
				})
			}
		}

		// Check if this window has the minimal cost
		if len(windowPlan) > 0 && totalCost < minCost {
			minCost = totalCost
			bestPlan = make(api.Rates, len(windowPlan))
			copy(bestPlan, windowPlan)
			bestPlan.Sort()
		}
	}

	t.log.TRACE.Printf("findOptimalContinuousWindow: checked %d windows, bestPlan has %d slots, minCost=%.3f",
		windowsChecked, len(bestPlan), minCost)

	// Merge individual slots into a single continuous slot with weighted average price
	if len(bestPlan) > 0 {
		// Calculate weighted average price per kWh using the already computed minCost
		avgPrice := minCost / effectiveDuration.Hours()

		mergedSlot := api.Rate{
			Start: bestPlan[0].Start,
			End:   bestPlan[len(bestPlan)-1].End,
			Value: avgPrice,
		}

		t.log.TRACE.Printf("findOptimalContinuousWindow: merged plan start=%v, end=%v, avgPrice=%.3f/kWh, totalCost=%.3f",
			mergedSlot.Start.Round(time.Second), mergedSlot.End.Round(time.Second), avgPrice, minCost)

		bestPlan = api.Rates{mergedSlot}
	}

	return bestPlan, minCost
}

// Plan creates a charging plan based on the configured or passed-in mode
// supports a continuous boolean flag to use single cheapest window mode
func (t *Planner) Plan(requiredDuration, precondition time.Duration, targetTime time.Time, continuous ...bool) api.Rates {
	if t == nil || requiredDuration <= 0 {
		return nil
	}

	useContinuous := len(continuous) > 0 && continuous[0]

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

	// consume remaining time
	if t.clock.Until(targetTime) <= requiredDuration {
		return t.continuousPlan(rates, latestStart, targetTime)
	}

	// rates are by default sorted by date, oldest to newest
	last := rates[len(rates)-1].End

	// reduce planning horizon to available rates
	if targetTime.After(last) {
		durationAfterRates := targetTime.Sub(last)
		if durationAfterRates >= requiredDuration {
			return nil
		}

		// need to use some of the available slots
		t.log.DEBUG.Printf("target time beyond available slots- reducing plan horizon from %v to %v",
			requiredDuration.Round(time.Second), durationAfterRates.Round(time.Second))

		targetTime = last
		requiredDuration -= durationAfterRates

		// recalculate latestStart after adjusting targetTime and requiredDuration
		latestStart = targetTime.Add(-requiredDuration)
		if latestStart.Before(t.clock.Now()) {
			latestStart = t.clock.Now()
		}
	}

	// use continuous window mode if selected
	if useContinuous {
		t.log.TRACE.Printf("using continuous mode: requiredDuration=%v, precondition=%v", requiredDuration, precondition)

		effectiveDuration := requiredDuration
		if precondition > 0 {
			effectiveDuration -= precondition
		}

		preCondWindow := targetTime.Add(-precondition)
		t.log.TRACE.Printf("searching optimal window: effectiveDuration=%v, preCondWindow=%v", effectiveDuration, preCondWindow)

		plan, cost := t.findOptimalContinuousWindow(rates, effectiveDuration, preCondWindow)

		if plan == nil {
			t.log.TRACE.Printf("no optimal window found, falling back to continuous plan")
			return t.continuousPlan(rates, latestStart, targetTime)
		}

		t.log.TRACE.Printf("found optimal window: start=%v, end=%v, cost=%.3f", plan[0].Start, plan[0].End, cost)

		// add preconditioning at the end
		if precondition > 0 {
			preCondStart := targetTime.Add(-precondition)
			preCondPlan := t.continuousPlan(rates, preCondStart, targetTime)
			t.log.TRACE.Printf("adding preconditioning: start=%v, end=%v", preCondStart, targetTime)
			plan = append(plan, preCondPlan...)
		}

		// sort plan by time
		plan.Sort()

		return plan
	}

	// default mode: cheapest combination of slots
	slices.SortStableFunc(rates, sortByCost)

	rates, adjusted := splitPreconditionSlots(rates, precondition, targetTime)

	// sort rates by price and time
	slices.SortStableFunc(rates, sortByCost)

	plan := t.plan(rates, requiredDuration, targetTime)

	// correct plan slots to show original, non-adjusted prices
	for i, r := range plan {
		if rr, err := adjusted.At(r.Start); err == nil {
			plan[i].Value = rr.Value
		}
	}

	// sort plan by time
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
