package planner

import (
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/jinzhu/copier"
	"golang.org/x/exp/slices"
)

// Planner plans a series of charging slots for a given (variable) tariff
type Planner struct {
	log    *util.Logger
	clock  clock.Clock // mockable time
	tariff api.Tariff
}

// New creates a price planner
func New(log *util.Logger, tariff api.Tariff) *Planner {
	return &Planner{
		log:    log,
		clock:  clock.New(),
		tariff: tariff,
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
		if source.Start.After(targetTime) || source.Start.Equal(targetTime) || source.End.Before(t.clock.Now()) {
			continue
		}

		var slot api.Rate
		if err := copier.Copy(&slot, source); err != nil {
			panic(err)
		}

		// adjust slot start and end
		if slot.Start.Before(t.clock.Now()) {
			slot.Start = t.clock.Now()
		}
		if slot.End.After(targetTime) {
			slot.End = targetTime
		}

		slotDuration := slot.End.Sub(slot.Start)
		requiredDuration -= slotDuration

		// slot covers more than we need, so lets start late
		if requiredDuration < 0 {
			slot.Start = slot.Start.Add(-requiredDuration)
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

func (t *Planner) Unit() string {
	if t.tariff == nil {
		return ""
	}
	return t.tariff.Unit()
}

// Plan creates a lowest-cost charging plan, considering edge conditions
func (t *Planner) Plan(requiredDuration time.Duration, targetTime time.Time) (api.Rates, error) {
	if t == nil || requiredDuration <= 0 {
		return nil, nil
	}

	// calculate start time
	latestStart := targetTime.Add(-requiredDuration)

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
	if t.clock.Now().After(latestStart) || t.clock.Now().Equal(latestStart) {
		requiredDuration = t.clock.Until(targetTime)
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

	return t.plan(rates, requiredDuration, targetTime), nil
}
