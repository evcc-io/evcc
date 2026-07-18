package planner

import (
	"slices"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
)

// Planner plans a series of charging slots for a given (variable) tariff
type Planner struct {
	log    *util.Logger
	clock  clock.Clock // mockable time
	tariff api.Tariff

	overrunWarned time.Time // target last warned as unreachable, avoids per-cycle spam
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
func optimalPlan(rates api.Rates, requiredDuration time.Duration, targetTime time.Time) api.Rates {
	return planCapped(rates, requiredDuration, targetTime, nil, 0, 0)
}

// planCapped builds a lowest-cost plan; with avail==nil each slot is full power
// (duration accounting), else min(maxPower, avail) and skipped below minPower.
func planCapped(rates api.Rates, requiredDuration time.Duration, targetTime time.Time, avail func(time.Time) float64, maxPower, minPower float64) api.Rates {
	plan := make(api.Rates, 0, int64(requiredDuration)/int64(tariff.SlotDuration)+3)

	for _, slot := range rates {
		slotDuration := slot.End.Sub(slot.Start)

		// per-slot power factor: full unless a capacity cap applies
		factor := 1.0
		if avail != nil && maxPower > 0 {
			p := avail(slot.Start)
			if p < minPower {
				continue // semi-continuous: charge >= min or not at all
			}
			factor = min(maxPower, p) / maxPower
		}

		requiredDuration -= time.Duration(float64(slotDuration) * factor)

		// slot covers more than we need, so shorten it
		if requiredDuration < 0 {
			trim := time.Duration(float64(-requiredDuration) / factor)
			// the first (if not single) slot should start as late as possible
			if IsFirst(slot, plan) && len(plan) > 0 {
				slot.Start = slot.Start.Add(trim)
			} else {
				slot.End = slot.End.Add(-trim)
			}
			requiredDuration = 0
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
func continuousPlan(rates api.Rates, start, end time.Time) api.Rates {
	res := clampRates(rates, start, end)

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

	now := t.clock.Now().Truncate(time.Second)

	latestStart := targetTime.Add(-requiredDuration)
	if latestStart.Before(now) {
		// goal missed: required duration no longer fits before the target, so
		// charging will overrun. warn once per target to avoid per-cycle spam.
		if !t.overrunWarned.Equal(targetTime) {
			t.log.WARN.Printf("planner: target not reachable in time - need %v but only %v until %v",
				requiredDuration.Round(time.Second), t.clock.Until(targetTime).Round(time.Second), targetTime.Local())
			t.overrunWarned = targetTime
		}
		latestStart = now
		targetTime = latestStart.Add(requiredDuration)
	} else {
		t.overrunWarned = time.Time{}
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
		t.log.DEBUG.Printf("planner: no rates available (count=%d, err=%v)- falling back to simple plan", len(rates), err)
		return simplePlan
	}

	t.log.TRACE.Printf("planner: %d rates available from %v to %v",
		len(rates), rates[0].Start.Local(), rates[len(rates)-1].End.Local())

	// consume remaining time
	if t.clock.Until(targetTime) <= requiredDuration {
		t.log.DEBUG.Printf("planner: insufficient time until target- charging continuously from now")
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

		targetTime = last
		requiredDuration -= durationAfterRates
		precondition = max(precondition-durationAfterRates, 0)
	}

	rates = clampRates(rates, now, targetTime)

	// check if rate coverage is sufficient for planning
	if len(rates) == 0 || Duration(rates) < requiredDuration {
		t.log.DEBUG.Printf("planner: rate coverage in [%v,%v] insufficient for required duration %v- falling back to simple plan",
			now.Local(), targetTime.Local(), requiredDuration.Round(time.Second))
		return simplePlan
	}

	// don't precondition longer than charging duration
	precondition = min(precondition, requiredDuration)

	// reduce target time by precondition duration
	targetTime = targetTime.Add(-precondition)

	// separate precond rates, to be appended to plan afterwards
	var precond api.Rates
	if precondition > 0 {
		rates, precond = splitPreconditionSlots(rates, targetTime)

		// reduce required duration by precondition, skip planning if required
		requiredDuration = max(requiredDuration-precondition, 0)
		if requiredDuration == 0 {
			return precond
		}
	}

	// create plan unless only precond slots remaining
	var plan api.Rates
	if continuous {
		// find cheapest continuous window
		plan = findContinuousWindow(rates, requiredDuration, targetTime)
	} else {
		// sort rates by price and time
		slices.SortStableFunc(rates, sortByCost)

		plan = optimalPlan(rates, requiredDuration, targetTime)

		// sort plan by time
		plan.Sort()
	}

	// re-append precondition slots
	plan = append(plan, precond...)

	return plan
}

func splitPreconditionSlots(rates api.Rates, preCondStart time.Time) (api.Rates, api.Rates) {
	var res, precond api.Rates

	for _, r := range rates {
		if !r.End.After(preCondStart) {
			res = append(res, r)
			continue
		}

		// split slot
		if r.Start.Before(preCondStart) {
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

		precond = append(precond, r)
	}

	return res, precond
}
