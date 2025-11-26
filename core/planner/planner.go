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
	var result api.Rates
	for _, r := range rates {
		// skip slots completely outside window
		if !r.End.After(start) || !r.Start.Before(end) {
			continue
		}

		slot := r
		adjustSlotStart(&slot, start)
		adjustSlotEnd(&slot, end)

		// only add slots with valid duration
		if isValidSlot(slot) {
			result = append(result, slot)
		}
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
		}

		// only add slots with valid duration
		if !isValidSlot(slot) {
			continue
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
// Returns the selected rates.
func (t *Planner) findContinuousWindow(rates api.Rates, effectiveDuration time.Duration, targetTime time.Time) api.Rates {
	bestCost := math.MaxFloat64
	var bestWindow api.Rates

	for i := range rates {
		windowEnd := rates[i].Start.Add(effectiveDuration)

		if windowEnd.After(targetTime) {
			break
		}

		// Collect all slots that fall within [Start, Start+effectiveDuration]
		var window api.Rates
		var duration time.Duration

		for j := i; j < len(rates) && duration < effectiveDuration; j++ {
			slot := rates[j]

			// slot partially or completely within window?
			if slot.Start.Before(windowEnd) {
				// trim end if necessary
				if slot.End.After(windowEnd) {
					slot.End = windowEnd
				}

				// only add slots with valid duration
				if isValidSlot(slot) {
					window = append(window, slot)
					duration += slot.End.Sub(slot.Start)
				}
			}
		}

		// only consider complete windows
		if duration < effectiveDuration {
			continue
		}

		// Calculate cost
		var cost float64
		for _, slot := range window {
			slotDur := slot.End.Sub(slot.Start)
			cost += float64(slotDur) * slot.Value
		}

		// Prefer later start if equal cost
		if cost <= bestCost {
			bestCost = cost
			bestWindow = window
		}
	}

	return bestWindow
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

	now := t.clock.Now().Truncate(time.Second)

	latestStart := targetTime.Add(-requiredDuration)
	if latestStart.Before(now) {
		latestStart = now
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

	// consume remaining time if total time until target is insufficient; regardless of tariff data availability
	if t.clock.Until(targetTime) <= requiredDuration {
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

		// Time after rates will be charged anyway, covers the precondition requirement
		targetTime = last
		requiredDuration -= durationAfterRates
		precondition = max(precondition-durationAfterRates, 0)
	}

	// filter rates to planning window early for performance
	rates = filterRates(rates, now, targetTime)

	// separate precond rates, to be appended to plan afterwards
	var precond api.Rates
	if precondition > 0 {
		// don't precondition longer than charging duration
		precondition = min(precondition, requiredDuration)

		rates, precond = splitAndAdjustPrecondition(rates, targetTime, precondition)

		// reduce required duration by precondition, skip planning if required
		requiredDuration = max(requiredDuration-precondition, 0)
		if requiredDuration == 0 {
			return precond
		}

		targetTime = targetTime.Add(-precondition)
		// chargingRates filtered by split (End <= preCondStart = new targetTime)
	}

	// create plan unless only precond slots remaining
	var plan api.Rates
	if continuous {
		// check if available tariff slots span is sufficient for sliding window algorithm
		// verify that actual tariff data covers enough duration (may have gaps or start late)
		if len(rates) > 0 {
			start := rates[0].Start
			if start.Before(now) {
				start = now
			}
			end := rates[len(rates)-1].End
			if end.After(targetTime) {
				end = targetTime
			}

			// available window too small for sliding window - charge continuously from now to target
			if end.Sub(start) < requiredDuration {
				return continuousPlan(append(rates, precond...), now, targetTime.Add(precondition))
			}
		}
		// find cheapest continuous window
		plan = t.findContinuousWindow(rates, requiredDuration, targetTime)
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

	// Use filterRates to trim the precondition window exactly
	precond = filterRates(precond, preCondStart, targetTime)

	// If we don't have enough duration, extend from chargingRates
	var total time.Duration
	for _, p := range precond {
		total += p.End.Sub(p.Start)
	}

	if deficit := precondition - total; deficit > 0 {
		// Prepend slots from chargingRates to fill the gap
		extendStart := preCondStart.Add(-deficit)
		extension := filterRates(chargingRates, extendStart, preCondStart)
		precond = append(extension, precond...)
	}

	return chargingRates, precond
}
