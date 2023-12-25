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

// WithClock sets a mockable clock
func WithClock(clock clock.Clock) func(t *Planner) {
	return func(t *Planner) {
		t.clock = clock
	}
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
		if source.Start.After(targetTime) || source.Start.Equal(targetTime) || source.End.Before(t.clock.Now()) {
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
		if r.End.Before(start) || r.End.Equal(start) {
			continue
		}

		// slot after continuous plan
		if r.Start.After(end) || r.Start.Equal(end) {
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

func (t *Planner) Plan(requiredDuration time.Duration, targetTime time.Time) (api.Rates, error) {
	if t == nil || requiredDuration <= 0 {
		return nil, nil
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
		return simplePlan, nil
	}

	rates, err := t.tariff.Rates()

	// treat like normal target charging if we don't have rates
	if len(rates) == 0 || err != nil {
		return simplePlan, err
	}

	// consume remaining time
	if t.clock.Until(targetTime) <= requiredDuration {
		return t.continuousPlan(rates, latestStart, targetTime), nil
	}

	// rates are by default sorted by date, oldest to newest
	last := rates[len(rates)-1].End

	// sort rates by price and time
	slices.SortStableFunc(rates, sortByCost)

	// reduce planning horizon to available rates
	if targetTime.After(last) {
		// there is enough time for charging after end of current rates
		durationAfterRates := targetTime.Sub(last)
		if durationAfterRates >= requiredDuration {
			return nil, nil
		}

		// need to use some of the available slots
		t.log.DEBUG.Printf("target time beyond available slots- reducing plan horizon from %v to %v",
			requiredDuration.Round(time.Second), durationAfterRates.Round(time.Second))

		targetTime = last
		requiredDuration -= durationAfterRates
	}

	plan := t.plan(rates, requiredDuration, targetTime)

	// sort plan by time
	plan.Sort()

	return plan, nil
}
