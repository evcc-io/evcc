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

// filterRates filters rates to the given time window and optionally adjusts boundary slots
func filterRates(rates api.Rates, start, end time.Time) api.Rates {
	if len(rates) == 0 {
		return nil
	}

	var result api.Rates
	for _, r := range rates {
		// skip slots completely outside window
		if !r.End.After(start) || !r.Start.Before(end) {
			continue
		}

		slot := r
		adjustSlotStart(&slot, start)
		adjustSlotEnd(&slot, end)

		result = append(result, slot)
	}

	return result
}

// plan creates a lowest-cost plan or required duration.
// It MUST already be established that:
// - rates are sorted in ascending order by cost and descending order by start time (prefer late slots)
// - rates are filtered to [now, targetTime] window by caller
func (t *Planner) plan(rates api.Rates, requiredDuration time.Duration, targetTime time.Time) api.Rates {
	var plan api.Rates

	// rates are already filtered by caller, so slots are guaranteed to be relevant

	for _, slot := range rates {
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
// - rates are filtered to [now, targetTime] window by caller
// Returns the selected rates and the total cost.
func (t *Planner) findContinuousWindow(validRates api.Rates, effectiveDuration time.Duration, targetTime time.Time) (api.Rates, float64) {
	if len(validRates) == 0 {
		return nil, 0
	}

	// rates are already filtered by caller, so slots are guaranteed to be relevant

	// Detect slot duration from first valid rate (make tests compatible)
	slotDuration := validRates[0].End.Sub(validRates[0].Start)
	slots := int(math.Ceil(float64(effectiveDuration) / float64(slotDuration)))

	bestSlot := -1
	bestCost := math.MaxFloat64

	// build prefix sum for fastest window cost calculation
	prefix := make([]float64, len(validRates)+1)
	for i := range validRates {
		prefix[i+1] = prefix[i] + validRates[i].Value
	}

	for i := range validRates {
		lastSlot := i + slots - 1
		windowEnd := validRates[i].Start.Add(effectiveDuration)

		if lastSlot >= len(validRates) || windowEnd.After(targetTime) {
			break
		}

		// use prefix sum to get total cost of this window
		sum := prefix[lastSlot+1] - prefix[i]

		// prefer later start if equal cost
		if sum <= bestCost {
			bestSlot = i
			bestCost = sum
		}
	}

	if bestSlot < 0 {
		return nil, 0
	}

	result := trimWindow(validRates[bestSlot:bestSlot+slots], effectiveDuration, targetTime)

	return result, bestCost
}

// Plan creates a continuous emergency charging plan
// ratest must be sorted by time
func continuousPlan(rates api.Rates, start, end time.Time) api.Rates {
	// filter and adjust rates to time window
	res := filterRates(rates, start, end)

	if len(res) == 0 {
		return api.Rates{
			api.Rate{
				Start: start,
				End:   end,
			},
		}
	}

	// prepend missing slot if rates don't start at plan start
	// required for scenarios where current time is before first available rate
	if res[0].Start.After(start) {
		res = slices.Insert(res, 0, api.Rate{
			Start: start,
			End:   res[0].Start,
		})
	}

	// append missing slot if rates don't extend to plan end
	// required for scenarios where target time is after last available rate
	if last := res[len(res)-1]; last.End.Before(end) {
		res = append(res, api.Rate{
			Start: last.End,
			End:   end,
		})
	}

	return res
}

func (t *Planner) Plan(requiredDuration time.Duration, targetTime time.Time, precondition time.Duration, continuous bool) api.Rates {
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
	}

	// filter rates to planning window early for performance
	rates = filterRates(rates, t.clock.Now(), targetTime)

	// separate precond rates, to be appended to plan afterwards
	var precond api.Rates

	if precondition > 0 {
		rates, precond = splitAndAdjustPrecondition(rates, targetTime, precondition)
		targetTime = targetTime.Add(-precondition)
		// chargingRates filtered by split (End <= preCondStart = new targetTime)
	}

	// reduce required duration by precondition
	requiredDuration = max(requiredDuration-precondition, 0)

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

func splitAndAdjustPrecondition(rates api.Rates, targetTime time.Time, precondition time.Duration) (api.Rates, api.Rates) {
	preCondStart := targetTime.Add(-precondition)

	var chargingRates, precond api.Rates

	// Split rates into charging and precondition periods
	for _, r := range rates {
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
	adjustSlotEnd(&precond[len(precond)-1], targetTime)

	// Calculate total duration
	var total time.Duration
	for _, p := range precond {
		total += p.End.Sub(p.Start)
	}

	// Adjust duration to match precondition exactly
	if diff := precondition - total; diff != 0 {
		if diff > 0 {
			// Deficit: prepend slots from chargingRates to fill the gap
			extendStart := precond[0].Start.Add(-diff)
			extension := filterRates(chargingRates, extendStart, precond[0].Start)
			precond = append(extension, precond...)
		} else {
			// Excess: trim first slot to start later
			trimSlot(&precond[0], -diff, false)
		}
	}

	return chargingRates, precond
}
