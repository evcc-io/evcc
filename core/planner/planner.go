package planner

import (
	"sort"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/jinzhu/copier"
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

// Plan creates a lowest-cost plan or required duration.
// It MUST already established that
// - rates are sorted in ascending order by cost and descending order by start time (prefer late slots)
// - target time and required duration are before end of rates
func (t *Planner) Plan(rates api.Rates, requiredDuration time.Duration, targetTime time.Time) api.Rates {
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
				t.log.ERROR.Print("slot end before start")
			}
		}

		plan = append(plan, slot)
		t.log.TRACE.Printf("  slot from: %v to %v cost %.3f", slot.Start.Round(time.Second).Local(), slot.End.Round(time.Second).Local(), slot.Price)

		// we found all necessary slots
		if requiredDuration == 0 {
			break
		}
	}

	return plan
}

// Active determines if current slot should be used for charging for a total required duration until target time
func (t *Planner) Active(requiredDuration time.Duration, targetTime time.Time) (time.Time, time.Time, bool, error) {
	if t == nil || requiredDuration <= 0 {
		return time.Time{}, time.Time{}, false, nil
	}

	// calculate start time
	latestStart := targetTime.Add(-requiredDuration)
	afterStart := t.clock.Now().After(latestStart) || t.clock.Now().Equal(latestStart)
	beforeTarget := t.clock.Now().Before(targetTime)

	// target charging without tariff
	if t.tariff == nil || afterStart {
		return latestStart, time.Time{}, afterStart && beforeTarget, nil
	}

	rates, err := t.tariff.Rates()

	// treat like normal target charging if we don't have rates
	if len(rates) == 0 || err != nil {
		return latestStart, time.Time{}, afterStart && beforeTarget, err
	}

	// rates are by default sorted by date, oldest to newest
	last := rates[len(rates)-1].End

	// sort rates by price and time
	sort.Sort(rates)

	// reduce planning horizon to available rates
	if targetTime.After(last) {
		// there is enough time for charging after end of current rates
		durationAfterRates := targetTime.Sub(last)
		if durationAfterRates >= requiredDuration {
			return time.Time{}, time.Time{}, false, nil
		}

		// need to use some of the available slots
		t.log.DEBUG.Printf("target time beyond available slots- reducing plan horizon from %v to %v",
			requiredDuration.Round(time.Second), durationAfterRates.Round(time.Second))

		targetTime = last
		requiredDuration -= durationAfterRates
	}

	plan := t.Plan(rates, requiredDuration, targetTime)

	var activeSlot api.Rate
	var planStart time.Time
	var planDuration time.Duration
	var planCost float64

	for _, slot := range plan {
		slotDuration := slot.End.Sub(slot.Start)
		planDuration += slotDuration
		planCost += float64(slotDuration) / float64(time.Hour) * slot.Price

		if planStart.IsZero() || slot.Start.Before(planStart) {
			planStart = slot.Start
		}

		if (slot.Start.Before(t.clock.Now()) || slot.Start.Equal(t.clock.Now())) && slot.End.After(t.clock.Now()) {
			activeSlot = slot
		}
	}

	t.log.DEBUG.Printf("total plan duration: %v, cost: %.2f", planDuration.Round(time.Second), planCost)

	return planStart, activeSlot.End, !activeSlot.End.IsZero(), nil
}
