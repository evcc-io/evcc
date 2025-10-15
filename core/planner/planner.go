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

func (t *Planner) findOptimalContinuousWindow(rates api.Rates, effectiveDuration time.Duration, targetTime time.Time) (api.Rates, float64) {
	now := t.clock.Now()
	if len(rates) == 0 || effectiveDuration <= 0 {
		return nil, 0
	}

	rates.Sort() // sort slots by start time
	//t.log.DEBUG.Printf("findOptimalContinuousWindow: now=%v, targetTime=%v, effectiveDuration=%v", now, targetTime, effectiveDuration)

	// prepare all relevant points (start/end of all slots)
	points := make([]time.Time, 0, 2*len(rates))
	for _, r := range rates {
		points = append(points, r.Start, r.End)
	}
	points = append(points, now, targetTime)
	slices.SortFunc(points, func(a, b time.Time) int { return int(a.Sub(b)) })
	points = slices.Compact(points)

	//t.log.DEBUG.Printf("findOptimalContinuousWindow: evaluated points: %v", len(points))
	for i, p := range points {
		t.log.DEBUG.Printf("  point[%d]: %v", i, p)
	}

	type windowSlot struct {
		Start, End time.Time
		Value      float64
	}

	var bestPlan api.Rates
	minCost := math.Inf(1)

	left := 0
	right := 0
	currentCost := 0.0
	activeSlots := []windowSlot{}

	// sliding-window over all relevant time points
	for left < len(points) {
		windowStart := points[left]
		windowEnd := windowStart.Add(effectiveDuration)

		//t.log.DEBUG.Printf("  left=%d: window [%v, %v]", left, windowStart, windowEnd)

		// Allow windowEnd == targetTime
		if windowEnd.After(targetTime) {
			//t.log.DEBUG.Printf("    windowEnd after targetTime, breaking")
			break
		}

		// remove slots that fall out on the left of the window
		newActive := activeSlots[:0]
		for _, s := range activeSlots {
			if s.End.After(windowStart) {
				newActive = append(newActive, s)
			} else {
				duration := s.End.Sub(s.Start).Hours()
				currentCost -= s.Value * duration
				//t.log.DEBUG.Printf("    removing left slot [%v, %v], duration=%.2fh, cost adjustment=%.4f", s.Start, s.End, duration, -s.Value*duration)
			}
		}
		activeSlots = newActive

		// add slots that come into the window on the right
		for right < len(rates) && rates[right].Start.Before(windowEnd) {
			s := rates[right]
			//t.log.DEBUG.Printf("    evaluating rate[%d]: [%v, %v] value=%.2f", right, s.Start, s.End, s.Value)

			trimStart := s.Start
			if windowStart.After(trimStart) {
				trimStart = windowStart
			}
			trimEnd := s.End
			if windowEnd.Before(trimEnd) {
				trimEnd = windowEnd
			}

			if trimStart.Before(trimEnd) {
				slotDuration := trimEnd.Sub(trimStart).Hours()
				slotCost := s.Value * slotDuration
				activeSlots = append(activeSlots, windowSlot{
					Start: trimStart,
					End:   trimEnd,
					Value: s.Value,
				})
				currentCost += slotCost
				//t.log.DEBUG.Printf("      added to window [%v, %v], duration=%.2fh, cost=%.4f, total=%.4f", trimStart, trimEnd, slotDuration, slotCost, currentCost)
			} else {
				//t.log.DEBUG.Printf("      trimmed slot invalid (start >= end)")
			}
			right++
		}

		// check if this window has the minimal cost
		//t.log.DEBUG.Printf("    window cost: %.4f (current min: %.4f), activeSlots: %d", currentCost, minCost, len(activeSlots))
		if currentCost < minCost {
			minCost = currentCost
			bestPlan = make(api.Rates, len(activeSlots))
			for i, s := range activeSlots {
				bestPlan[i] = api.Rate{
					Start: s.Start,
					End:   s.End,
					Value: s.Value,
				}
			}
			bestPlan.Sort()
			//t.log.DEBUG.Printf("    NEW BEST: cost=%.4f with %d slots", minCost, len(bestPlan))
		}

		left++
	}

	//	t.log.DEBUG.Printf("findOptimalContinuousWindow result: bestPlan length=%d, minCost=%.4f", len(bestPlan), minCost)
	//	if bestPlan != nil {
	//		for i, slot := range bestPlan {
	//			t.log.DEBUG.Printf("  slot[%d]: [%v, %v] value=%.2f", i, slot.Start, slot.End, slot.Value)
	//		}
	//	}

	// Merge individual slots into a single continuous slot with weighted average price
	if len(bestPlan) > 0 {
		mergedSlot := api.Rate{
			Start: bestPlan[0].Start,
			End:   bestPlan[len(bestPlan)-1].End,
		}

		// Calculate weighted average price per kWh
		totalCost := 0.0
		totalDuration := 0.0
		for _, slot := range bestPlan {
			duration := slot.End.Sub(slot.Start).Hours()
			totalCost += slot.Value * duration
			totalDuration += duration
		}

		if totalDuration > 0 {
			mergedSlot.Value = totalCost / totalDuration
		}

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
	}

	// use continuous window mode if selected
	if useContinuous {
		effectiveDuration := requiredDuration
		if precondition > 0 {
			effectiveDuration -= precondition
		}

		preCondWindow := targetTime.Add(-precondition)
		plan, _ := t.findOptimalContinuousWindow(rates, effectiveDuration, preCondWindow)

		if plan == nil {
			return t.continuousPlan(rates, latestStart, targetTime)
		}

		// add preconditioning at the end
		if precondition > 0 {
			preCondStart := targetTime.Add(-precondition)
			preCondPlan := t.continuousPlan(rates, preCondStart, targetTime)
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
