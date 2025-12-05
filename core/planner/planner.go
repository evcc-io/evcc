package planner

import (
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
// It MUST already be established that:
// - rates are sorted in ascending order by cost and descending order by start time (prefer late slots)
// - rates are filtered to [now, targetTime] window by caller
func (t *Planner) plan(rates api.Rates, requiredDuration time.Duration, targetTime time.Time) api.Rates {
	var plan api.Rates

	for _, slot := range rates {
		if requiredDuration <= 0 {
			break
		}

		slotDuration := slot.End.Sub(slot.Start)

		// slot covers more than we need, so shorten it
		if slotDuration > requiredDuration {
			trimSlot(&slot, slotDuration-requiredDuration, !(IsFirst(slot, plan) && len(plan) > 0))
			requiredDuration = 0
		} else {
			requiredDuration -= slotDuration
		}

		plan = append(plan, slot)
	}

	return plan
}

// filterRates filters rates to the given time window and adjusts boundary slots
func filterRates(rates api.Rates, start, end time.Time) api.Rates {
	res := make(api.Rates, 0, len(rates))

	for _, r := range rates {
		// slot before continuous plan
		if !r.End.After(start) {
			continue
		}

		// slot after continuous plan
		if !r.Start.Before(end) {
			continue
		}

		// calculate adjusted bounds
		adjustedStart := r.Start
		if adjustedStart.Before(start) {
			adjustedStart = start
		}

		adjustedEnd := r.End
		if adjustedEnd.After(end) {
			adjustedEnd = end
		}

		// skip if adjustment would create invalid slot
		if !adjustedEnd.After(adjustedStart) {
			continue
		}

		slot := r
		slot.Start = adjustedStart
		slot.End = adjustedEnd
		res = append(res, slot)
	}

	return res
}

// continuousPlan creates a continuous emergency charging plan
func continuousPlan(rates api.Rates, start, end time.Time) api.Rates {
	res := filterRates(rates, start, end)

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

func (t *Planner) Plan(requiredDuration time.Duration, precondition time.Duration, targetTime time.Time, continuous bool) api.Rates {
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

	// consume remaining time
	if t.clock.Until(targetTime) <= requiredDuration {
		return continuousPlan(rates, latestStart, targetTime)
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
		plan = findContinuousWindow(rates, requiredDuration, targetTime)
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

	var res, precond api.Rates

	for _, r := range rates {
		if !r.End.After(preCondStart) {
			res = append(res, r)
			continue
		}
		precond = append(precond, r)
	}

	precond = filterRates(precond, preCondStart, targetTime)

	var total time.Duration
	for _, p := range precond {
		total += p.End.Sub(p.Start)
	}

	if deficit := precondition - total; deficit > 0 {
		extendStart := preCondStart.Add(-deficit)
		extension := filterRates(res, extendStart, preCondStart)
		precond = append(extension, precond...)
	}

	return res, precond
}
