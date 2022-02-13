package core

import (
	"sort"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

const (
	chargeEfficiency   = 0.95
	hysteresisDuration = 5 * time.Minute
)

// RatesByPrice implements sort.Interface based on price
type RatesByPrice []api.Rate

func (a RatesByPrice) Len() int {
	return len(a)
}

func (a RatesByPrice) Less(i, j int) bool {
	if a[i].Price == a[j].Price {
		return a[i].Start.After(a[j].Start)
	}
	return a[i].Price < a[j].Price
}

func (a RatesByPrice) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

type Planner struct {
	log    *util.Logger
	clock  clock.Clock // mockable time
	tariff api.Tariff
}

func NewPlanner(log *util.Logger, tariff api.Tariff) *Planner {
	clock := clock.New()
	return &Planner{
		log:    log,
		clock:  clock,
		tariff: tariff,
	}
}

func (t *Planner) PlanActive(requiredDuration time.Duration, targetTime time.Time) (bool, error) {
	if targetTime.Before(t.clock.Now()) || requiredDuration <= 0 {
		return false, nil
	}

	data, err := t.tariff.Rates()
	if err != nil {
		return false, err
	}

	// rates are by default sorted by date, oldest to newest
	last := data[len(data)-1].End

	// sort rates by price
	sort.Sort(RatesByPrice(data))

	requiredDuration = time.Duration(float64(requiredDuration) / chargeEfficiency)

	// Save same duration until next price info update
	if targetTime.After(last) {
		duration_old := requiredDuration
		requiredDuration = time.Duration(float64(requiredDuration) * float64(time.Until(last)) / float64(time.Until(targetTime)))
		t.log.DEBUG.Printf("reduced duration from %s to %s until got new priceinfo after %s\n", duration_old.Round(time.Minute), requiredDuration.Round(time.Minute), last.Round(time.Minute))
	}

	t.log.DEBUG.Printf("charge duration: %s, end: %v, find best prices:\n", requiredDuration.Round(time.Minute), targetTime.Round(time.Minute))

	var cheapActive bool
	var plannedSlots, currentSlot int
	var plannedDuration time.Duration

	for _, slot := range data {
		// slot not relevant
		if slot.Start.After(targetTime) || slot.End.Before(t.clock.Now()) {
			continue
		}

		// current slot
		if slot.Start.Before(t.clock.Now()) && slot.End.After(t.clock.Now()) {
			slot.Start = t.clock.Now().Add(-1)
		}

		// slot ends after target time
		if slot.End.After(targetTime) {
			slot.End = targetTime.Add(1)
		}

		plannedSlots++
		plannedDuration += slot.End.Sub(slot.Start)

		t.log.TRACE.Printf("  Slot from: %v to %v price %f, timesum %s",
			slot.Start.Round(time.Second), slot.End.Round(time.Second),
			slot.Price, plannedDuration)

		// plan covers current slot
		if slot.Start.Before(t.clock.Now().Add(1)) && slot.End.After(t.clock.Now()) {
			cheapActive = true
			currentSlot = plannedSlots
			t.log.TRACE.Printf(" (now, slot number %v)", currentSlot)
		}

		// we found all necessary cheap slots to charge to targetSoC
		if plannedDuration >= requiredDuration {
			break
		}
	}

	// delay start of most expensive slot as long as possible
	if currentSlot == plannedSlots && plannedSlots > 1 && plannedDuration > requiredDuration+hysteresisDuration {
		t.log.DEBUG.Printf("cheap timeslot, delayed for %s\n", (plannedDuration - requiredDuration).Round(time.Minute))
		cheapActive = false
	}

	return cheapActive, nil
}
