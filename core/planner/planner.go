
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

	result := validRates[bestSlot : bestSlot+slots]

	// Trim if allocated duration exceeds required duration
	totalDuration := time.Duration(slots) * slotDuration
	if totalDuration > effectiveDuration {
		excess := totalDuration - effectiveDuration

		// Single slot: trim end (start early)
		// Multiple slots: trim first slot start (start late)
		if len(result) == 1 {
			result[0].End = result[0].End.Add(-excess)
		} else {
			result[0].Start = result[0].Start.Add(excess)
		}
	}

	return result, bestCost
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

	if len(res) == 0 {
		return []api.Rate{{
			Start: start,
			End:   end,
		}}
	}

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
	if t.clock.Until(targetTime) <= requiredDuration {
		return t.continuousPlan(rates, latestStart, targetTime)
	}

	// cut off all rates after target time
	for i := 1; i < len(rates); i++ {
		if !rates[i].Start.Before(targetTime) {
			rates = rates[:i]
			break
		}
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
		precondition = max(precondition-durationAfterRates, 0)
	}

	// reduce target time by precondition duration
	originalTargetTime := targetTime
	targetTime = targetTime.Add(-precondition)
	requiredDuration = max(requiredDuration-precondition, 0)

	// separate precond rates, to be appended to plan afterwards
	var precond api.Rates
	if precondition > 0 {
		rates, precond = splitPreconditionSlots(rates, targetTime)

		// Trim precondition slots to exactly match precondition duration and add excess to required
		if len(precond) > 0 {
			if excess := originalTargetTime.Add(-precondition).Sub(precond[0].Start); excess > 0 {
				precond[0].Start, requiredDuration = precond[0].Start.Add(excess), requiredDuration+excess
			}
			if lastIdx := len(precond) - 1; precond[lastIdx].End.After(originalTargetTime) {
				requiredDuration += precond[lastIdx].End.Sub(originalTargetTime)
				precond[lastIdx].End = originalTargetTime
			}
		}
	}

	// check if available rates span is sufficient for sliding window
	if continuous && requiredDuration > 0 && len(rates) > 0 {
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
			return t.continuousPlan(append(rates, precond...), now, targetTime.Add(precondition))
		}
	}

	// create plan unless only precond slots remaining
	var plan api.Rates
	if requiredDuration > 0 {
		if continuous {
			// continuous mode: find cheapest continuous window
			plan, _ = t.findContinuousWindow(rates, requiredDuration, targetTime)
		} else {
			// dispersed mode: find cheapest combination of slots
			// sort rates by price and time
			slices.SortStableFunc(rates, sortByCost)
			plan = t.plan(rates, requiredDuration, targetTime)

			// sort plan by time
			plan.Sort()
		}
	}

	// re-append precondition slots
	plan = append(plan, precond...)

	return plan
}

func splitPreconditionSlots(rates api.Rates, preCondStart time.Time) (api.Rates, api.Rates) {
	var res, precond api.Rates

	for _, r := range slices.Clone(rates) {
		if !r.End.After(preCondStart) {
			res = append(res, r)
			continue
		}

		precond = append(precond, r)
	}

	return res, precond
}

