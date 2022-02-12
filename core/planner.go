package core

import (
	"sort"
	"sync"
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
	mux    sync.Mutex
}

func NewPlanner(log *util.Logger, tariff api.Tariff) *Planner {
	clock := clock.New()
	return &Planner{
		log:    log,
		clock:  clock,
		tariff: tariff,
	}
}

func (t *Planner) isCheapSlotNow(duration time.Duration, end time.Time) (bool, error) {
	cheapSlotNow := false
	cntExpectedSlots := 0
	curSlotNr := 0
	var pData api.Rate
	var sum time.Duration

	data, err := t.tariff.Rates()
	if err != nil {
		return false, err
	}
	last := data[len(data)-1].End

	duration = time.Duration(float64(duration) / chargeEfficiency)

	// Save same duration until next price info update
	if end.After(last) {
		duration_old := duration
		duration = time.Duration(float64(duration) * float64(time.Until(last)) / float64(time.Until(end)))
		t.log.DEBUG.Printf("reduced duration from %s to %s until got new priceinfo after %s\n", duration_old.Round(time.Minute), duration.Round(time.Minute), last.Round(time.Minute))
	}

	t.log.DEBUG.Printf("charge duration: %s, end: %v, find best prices:\n", duration.Round(time.Minute), end.Round(time.Second))

	sort.Sort(RatesByPrice(data))

	for i := 0; i < len(data); i++ {
		pData = data[i]

		if pData.Start.Before(t.clock.Now()) && pData.End.After(t.clock.Now()) { // current slot
			pData.Start = t.clock.Now()
		}

		// timeslot ends after charge finish time
		if pData.End.After(end) {
			pData.End = end
		}

		if !(pData.Start.Before(end)) || pData.End.Before(t.clock.Now()) { // charge should ends before || old data
			continue
		}

		cntExpectedSlots++
		sum += pData.End.Sub(pData.Start)
		t.log.TRACE.Printf("  Slot from: %v to %v price %f, timesum %s",
			pData.Start.Round(time.Second), pData.End.Round(time.Second),
			pData.Price, sum)

		// current timeslot is a cheap one
		if !pData.Start.After(t.clock.Now()) && pData.End.After(t.clock.Now()) && duration > 0 {
			cheapSlotNow = true
			curSlotNr = cntExpectedSlots
			t.log.TRACE.Printf(" (now, slot number %v)", curSlotNr)
		}

		// we found all necessary cheap slots to charge to targetSoC
		if sum > duration {
			break
		}
	}

	var cheapactive bool
	if cheapSlotNow {
		if curSlotNr == cntExpectedSlots { // delay most expensiv slot if not last slot
			if cntExpectedSlots == 1 { // last slot
				t.log.DEBUG.Printf("continue charging in last slot\n")
				cheapactive = true
			} else { // expensiv and not last slot, delay
				if sum > duration+hysteresisDuration {
					t.log.DEBUG.Printf("cheap timeslot, delayed for %s\n", (sum - duration).Round(time.Minute))
					cheapactive = false
				} else {
					t.log.DEBUG.Printf("charing in most expensiv timeslot after delay")
					cheapactive = true
				}
			}
		} else { // not most expensiv slot
			t.log.DEBUG.Printf("cheap timeslot, charging...\n")
			cheapactive = true
		}
	}

	return cheapactive, nil
}

func (t *Planner) PlanActive(duration time.Duration, end time.Time) (bool, error) {
	t.mux.Lock()
	defer t.mux.Unlock()

	if end.Before(t.clock.Now()) {
		return false, nil
	}

	return t.isCheapSlotNow(duration, end)
}
