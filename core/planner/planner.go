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
// 3. Returns the window with minimal cost
//
// Returns: best plan and the total cost
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
		// Use <= to prefer the latest window when costs are equal (late charging)
		if len(windowPlan) > 0 && totalCost <= minCost {
			minCost = totalCost
			bestPlan = make(api.Rates, len(windowPlan))
			copy(bestPlan, windowPlan)
			bestPlan.Sort()
		}
	}

	t.log.TRACE.Printf("findOptimalContinuousWindow: checked %d windows, bestPlan has %d slots, minCost=%.3f",
		windowsChecked, len(bestPlan), minCost)

	// Return plan with actual prices per slot
	if len(bestPlan) > 0 {
		t.log.TRACE.Printf("findOptimalContinuousWindow: plan start=%v, end=%v, slots=%d, totalCost=%.3f",
			bestPlan[0].Start.Round(time.Second), bestPlan[len(bestPlan)-1].End.Round(time.Second), len(bestPlan), minCost)
	}

	return bestPlan, minCost
}

// applyPreconditionToPlan adds preconditioning to an existing plan and adjusts timing.
// This is used by both continuous and cost-minimized planning modes.
func (t *Planner) applyPreconditionToPlan(plan api.Rates, rates api.Rates, effectiveDuration, precondition time.Duration, targetTime time.Time, useContinuous bool) api.Rates {
	// Apply "start as late as possible" logic by shifting individual slots
	// Each slot is only shifted within its own rate window
	if len(plan) > 0 && precondition > 0 && useContinuous {
		// Process each slot individually
		for i := range plan {
			slot := &plan[i]
			
			// Find the rate(s) that contain this slot
			for _, rate := range rates {
				// Check if this rate contains the start of the slot
				if rate.Start.After(slot.Start) {
					continue
				}
				if rate.End.Before(slot.Start) || rate.End.Equal(slot.Start) {
					continue
				}
				
				// This rate contains the start of our slot
				// Calculate how much space is available within this rate
				availableSpace := rate.End.Sub(slot.End)
				
				if availableSpace > 0 {
					// We can shift this slot later within its rate window
					shift := availableSpace
					
					// But we need to ensure the slot doesn't overlap with the next slot
					if i+1 < len(plan) {
						nextSlot := plan[i+1]
						maxShiftUntilNextSlot := nextSlot.Start.Sub(slot.End)
						if maxShiftUntilNextSlot < shift {
							shift = maxShiftUntilNextSlot
						}
					}
					
					// Apply the shift to this slot
					if shift > 0 {
						slot.Start = slot.Start.Add(shift)
						slot.End = slot.End.Add(shift)
						t.log.TRACE.Printf("shifted slot %d: shift=%v, new start=%v, new end=%v", 
							i, shift, slot.Start, slot.End)
					}
				}
				
				break // Found the containing rate, no need to continue
			}
		}
	}

	// Add preconditioning at the end
	if precondition > 0 {
		preCondStart := targetTime.Add(-precondition)
		preCondPlan := t.continuousPlan(rates, preCondStart, targetTime)
		t.log.TRACE.Printf("adding preconditioning: start=%v, end=%v", preCondStart, targetTime)
		plan = append(plan, preCondPlan...)
	}

	// Sort plan by time
	plan.Sort()

	return plan
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
		preCondWindow := targetTime
		if precondition > 0 {
			effectiveDuration -= precondition
			preCondWindow = targetTime.Add(-precondition)
		}

		// if entire duration is preconditioning, skip planning
		if effectiveDuration <= 0 {
			return t.continuousPlan(rates, targetTime.Add(-precondition), targetTime)
		}

		t.log.TRACE.Printf("searching optimal window: effectiveDuration=%v, preCondWindow=%v", effectiveDuration, preCondWindow)

		plan, cost := t.findOptimalContinuousWindow(rates, effectiveDuration, preCondWindow)

		if plan == nil {
			t.log.TRACE.Printf("no optimal window found, falling back to continuous plan")
			return t.continuousPlan(rates, latestStart, targetTime)
		}

		t.log.TRACE.Printf("found optimal window: start=%v, end=%v, slots=%d, cost=%.3f",
			plan[0].Start, plan[len(plan)-1].End, len(plan), cost)

		// add preconditioning and adjust timing
		return t.applyPreconditionToPlan(plan, rates, effectiveDuration, precondition, targetTime, useContinuous)
	}

	// default mode: cheapest combination of slots
	effectiveDuration := requiredDuration
	preCondWindow := targetTime
	if precondition > 0 {
		effectiveDuration -= precondition
		preCondWindow = targetTime.Add(-precondition)
	}

	// if entire duration is preconditioning, skip planning
	if effectiveDuration <= 0 {
		return t.continuousPlan(rates, targetTime.Add(-precondition), targetTime)
	}

	slices.SortStableFunc(rates, sortByCost)

	plan := t.plan(rates, effectiveDuration, preCondWindow)

	// add preconditioning and adjust timing
	return t.applyPreconditionToPlan(plan, rates, effectiveDuration, precondition, targetTime, useContinuous)
}
