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
	ledger *Ledger      // shared circuit capacity, nil = unlimited
	owner  func() Owner // identifies the loadpoint in the ledger
}

// WithLedger shares circuit capacity with the other loadpoints' plans
func WithLedger(ledger *Ledger, owner func() Owner) func(*Planner) {
	return func(p *Planner) {
		p.ledger = ledger
		p.owner = owner
	}
}

// Reserve records the plan against the shared circuit capacity, an empty plan releases it
func (t *Planner) Reserve(plan api.Rates) {
	if t == nil || t.ledger == nil {
		return
	}
	t.ledger.Reserve(t.owner(), plan)
}

// usable returns the power available to the owner during the slot and the fraction
// of the slot's duration that counts against the required duration
func usable(slot api.Rate, maxPower float64, available func(api.Rate) float64) (float64, float64) {
	if available == nil || maxPower <= 0 {
		return maxPower, 1
	}
	power := min(maxPower, available(slot))
	return power, power / maxPower
}

// effectiveDuration returns the duration the rates provide at the owner's share
func effectiveDuration(rates api.Rates, maxPower float64, available func(api.Rate) float64) time.Duration {
	var res time.Duration
	for _, slot := range rates {
		_, share := usable(slot, maxPower, available)
		res += time.Duration(float64(slot.End.Sub(slot.Start)) * share)
	}
	return res
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
func optimalPlan(rates api.Rates, requiredDuration time.Duration, targetTime time.Time, maxPower float64, available func(api.Rate) float64) api.Rates {
	plan := make(api.Rates, 0, int64(requiredDuration)/int64(tariff.SlotDuration)+3)

	for _, slot := range rates {
		// a slot shared with other loadpoints counts only with the owner's share
		power, share := usable(slot, maxPower, available)
		if share <= 0 {
			continue
		}
		slot.Power = power

		slotDuration := time.Duration(float64(slot.End.Sub(slot.Start)) * share)
		requiredDuration -= slotDuration

		// slot covers more than we need, so shorten it
		if requiredDuration < 0 {
			excess := time.Duration(float64(-requiredDuration) / share)
			// the first (if not single) slot should start as late as possible
			if IsFirst(slot, plan) && len(plan) > 0 {
				slot.Start = slot.Start.Add(excess)
			} else {
				slot.End = slot.End.Add(-excess)
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

	// power the plan is executed with and the share of it left by outranking loadpoints
	var maxPower float64
	var available func(api.Rate) float64
	if t.ledger != nil {
		owner := t.owner()
		owner.Target = targetTime
		maxPower = owner.MaxPower

		available = func(slot api.Rate) float64 {
			if power := t.ledger.Available(owner, slot); power >= owner.MinPower {
				return power
			}
			return 0
		}
	}

	plan := t.plan(requiredDuration, precondition, targetTime, continuous, maxPower, available)

	// slots not planned against the ledger run at full power
	for i := range plan {
		if plan[i].Power == 0 {
			plan[i].Power = maxPower
		}
	}

	return plan
}

func (t *Planner) plan(requiredDuration, precondition time.Duration, targetTime time.Time, continuous bool, maxPower float64, available func(api.Rate) float64) api.Rates {
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
	if len(rates) == 0 || effectiveDuration(rates, maxPower, available) < requiredDuration {
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

		plan = optimalPlan(rates, requiredDuration, targetTime, maxPower, available)

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
