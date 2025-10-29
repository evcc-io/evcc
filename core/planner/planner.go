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

// trimSlot trims excess duration from a slot
// Single slot: trim end (start early)
// Multiple slots (first slot): trim start (start late)
func trimSlot(slot *api.Rate, excess time.Duration, isSingle bool) {
	if isSingle {
		slot.End = slot.End.Add(-excess)
	} else {
		slot.Start = slot.Start.Add(excess)
	}
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
			trimSlot(&slot, -requiredDuration, !(IsFirst(slot, plan) && len(plan) > 0))
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

// findContinuousWindow finds the cheapest continuous window of slots for the given duration.
// Returns the selected rates and the total cost.
func (t *Planner) findContinuousWindow(rates api.Rates, effectiveDuration time.Duration, targetTime time.Time) (api.Rates, float64) {
	now := t.clock.Now()

	// Filter rates that end after now and start before targetTime
	var validRates api.Rates
	for _, r := range rates {
		if r.End.After(now) && r.Start.Before(targetTime) {
			validRates = append(validRates, r)
		}
	}

	if len(validRates) == 0 {
		return nil, 0
	}

	// Detect slot duration from first valid rate (make tests compatible)
	slotDuration := validRates[0].End.Sub(validRates[0].Start)
	slots := int(math.Ceil(float64(effectiveDuration) / float64(slotDuration)))

	bestSlot := -1
	bestCost := math.MaxFloat64

	for i := range validRates {
		lastSlot := i + slots - 1
		windowEnd := validRates[i].Start.Add(effectiveDuration)
		if lastSlot >= len(validRates) || windowEnd.After(targetTime) {
			break
		}

		var sum float64
		for j := i; j <= lastSlot; j++ {
			sum += validRates[j].Value
		}

		// prefer later start if equal cost
		if sum <= bestCost {
			bestSlot = i
			bestCost = sum
		}
	}

	if bestSlot < 0 {
		return nil, 0
	}

	// edge case: target at non-slot boundary coinciding with optimal window
	result := trimAndAlignWindow(validRates[bestSlot:bestSlot+slots], effectiveDuration, targetTime, slotDuration)

	return result, bestCost
}

// trimAndAlignWindow adjusts a continuous window to match the target time and required duration.
// It trims and shifts as needed to ensure the plan aligns exactly with targetTime.
func trimAndAlignWindow(window api.Rates, effectiveDuration time.Duration, targetTime time.Time, slotDuration time.Duration) api.Rates {
	n := len(window)
	if n == 0 {
		return window
	}

	last := n - 1

	// trim the end to targetTime if needed
	if window[last].End.After(targetTime) {
		window[last].End = targetTime
	}

	// trim excess from the start if window is too long
	current := window[last].End.Sub(window[0].Start)
	if excess := current - effectiveDuration; excess > 0 {
		trimSlot(&window[0], excess, n == 1)
	}

	// shift forward if window ends slightly before targetTime (mid-slot alignment)
	if gap := targetTime.Sub(window[last].End); gap > 0 && gap < slotDuration {
		for i := range window {
			window[i].Start = window[i].Start.Add(gap)
			window[i].End = window[i].End.Add(gap)
		}
	}

	return window
}

// Plan creates a continuous emergency charging plan
// ratest must be sorted by time
func continuousPlan(rates api.Rates, start, end time.Time) api.Rates {
	res := make(api.Rates, 0, len(rates)+2)
	for _, r := range rates {
		// TODO do this outside, too?

		// slot before continuous plan
		if !r.End.After(start) {
			continue
		}

		// TODO this has already been done outside

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
		return api.Rates{
			api.Rate{
				Start: start,
				End:   end,
			},
		}
	}

	// TODO do we want to care for missing slots in the beginning?
	// if yes- do here or outside?

	// prepend missing slot
	if res[0].Start.After(start) {
		res = slices.Insert(res, 0, api.Rate{
			Start: start,
			End:   res[0].Start,
		})
	}

	// TODO isn't this already handled outside?

	// append missing slot
	if last := res[len(res)-1]; last.End.Before(end) {
		res = append(res, api.Rate{
			Start: last.End,
			End:   end,
		})
	}

	return res
}

func (t *Planner) Plan(requiredDuration, precondition time.Duration, targetTime time.Time, continuous bool) api.Rates {
	if t == nil || requiredDuration <= 0 {
		return nil
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

	// consume remaining time
	if t.clock.Until(targetTime) <= requiredDuration || precondition >= requiredDuration {
		return continuousPlan(rates, latestStart, targetTime)
	}

	// reduce planning horizon to available rates
	if last := rates[len(rates)-1].End; targetTime.After(last) {
		// there is enough time for charging after end of current rates
		durationAfterRates := targetTime.Sub(last)
		if durationAfterRates >= requiredDuration {
			return nil
		}

		// need to use some of the available slots
		t.log.DEBUG.Printf("planner: target time beyond available slots- reducing plan horizon from %v to %v",
			requiredDuration.Round(time.Second), durationAfterRates.Round(time.Second))

		targetTime = last
		requiredDuration -= durationAfterRates
		precondition = max(precondition-durationAfterRates, 0)
	} else {
		// performance: cut off all rates after target time
		for i := 1; i < len(rates); i++ {
			if !rates[i].Start.Before(targetTime) {
				rates = rates[:i]
				break
			}
		}
	}

	// reduce target time by precondition duration
	originalTargetTime := targetTime
	targetTime = targetTime.Add(-precondition)
	requiredDuration = max(requiredDuration-precondition, 0)

	// separate precond rates, to be appended to plan afterwards
	var precond api.Rates
	if precondition > 0 {
		rates, precond = splitAndAdjustPrecondition(rates, targetTime, precondition, originalTargetTime)
	}

	// create plan unless only precond slots remaining
	var plan api.Rates
	if continuous {
		// check if available rates span is sufficient for sliding window
		if len(rates) > 0 {
			now := t.clock.Now()
			start := rates[0].Start
			if start.Before(now) {
				start = now
			}
			end := rates[len(rates)-1].End
			if end.After(targetTime) {
				end = targetTime
			}

			// available window too small for sliding window - use continuous plan without preconditioning
			if end.Sub(start) < requiredDuration {
				return continuousPlan(append(rates, precond...), now, targetTime.Add(precondition))
			}
		}
		// find cheapest continuous window
		plan, _ = t.findContinuousWindow(rates, requiredDuration, targetTime)
	} else {
		// find cheapest combination of slots
		slices.SortStableFunc(rates, sortByCost)
		plan = t.plan(rates, requiredDuration, targetTime)

		plan.Sort()
	}

	// re-append adjusted precondition slots
	plan = append(plan, precond...)

	return plan
}

func splitAndAdjustPrecondition(rates api.Rates, preCondStart time.Time, precondition time.Duration, targetTime time.Time) (api.Rates, api.Rates) {
	var chargingRates, precond api.Rates

	// Split rates into charging and precondition periods
	for _, r := range slices.Clone(rates) {
		if !r.End.After(preCondStart) {
			chargingRates = append(chargingRates, r)
			continue
		}
		precond = append(precond, r)
	}

	if len(precond) == 0 {
		return chargingRates, precond
	}

	// Trim last slot to end exactly at target
	if precond[len(precond)-1].End.After(targetTime) {
		precond[len(precond)-1].End = targetTime
	}

	// Calculate total duration
	var total time.Duration
	for _, p := range precond {
		total += p.End.Sub(p.Start)
	}

	// Adjust duration to match precondition exactly
	if diff := precondition - total; diff != 0 {
		if diff > 0 {
			// Deficit: prepend slots from original rates to fill the gap
			extendStart := precond[0].Start.Add(-diff)
			var extension api.Rates
			for _, r := range rates {
				if r.End.Before(extendStart) || r.End.Equal(extendStart) {
					continue
				}
				if r.Start.After(precond[0].Start) || r.Start.Equal(precond[0].Start) {
					break
				}
				// Trim to fit the extension period
				slot := r
				if slot.Start.Before(extendStart) {
					slot.Start = extendStart
				}
				if slot.End.After(precond[0].Start) {
					slot.End = precond[0].Start
				}
				extension = append(extension, slot)
			}
			precond = append(extension, precond...)
		} else {
			// Excess: trim first slot to start later
			precond[0].Start = precond[0].Start.Add(-diff)
		}
	}

	return chargingRates, precond
}
