package core

import (
	"sort"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
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
	log         *util.Logger
	tariff      api.Tariff
	mux         sync.Mutex
	cheap       float64
	cheapactive bool
	last        time.Time
}

func NewPlanner(log *util.Logger, tariff api.Tariff) *Planner {
	return &Planner{
		log:    log,
		cheap:  0.2, // TODO
		tariff: tariff,
	}
}

func (t *Planner) isCheapSlotNow(duration time.Duration, end time.Time) (bool, error) {
	cheapSlotNow := false
	cntExpectedSlots := 0
	curSlotNr := 0
	var pi api.Rate
	var sum time.Duration

	data, err := t.tariff.Rates()
	if err != nil {
		return false, err
	}

	sort.Sort(RatesByPrice(data))

	for i := 0; i < len(data); i++ {
		pi = data[i]

		if pi.Start.Before(time.Now()) && pi.End.After(time.Now()) { // current slot
			pi.Start = time.Now()
		}

		if pi.End.Before(time.Now()) { // old data
			continue
		}

		if !(pi.Start.Before(end)) { // charge ends before
			continue
		}

		// timeslot already started
		pstart := pi.Start
		if pstart.Before(time.Now()) {
			pstart = time.Now()
		}

		// timeslot ends after charge finish time
		pend := pi.End
		if pend.After(end) {
			pend = end
		}

		cntExpectedSlots++
		delta := pend.Sub(pstart)
		sum += delta
		t.log.TRACE.Printf("  Slot from: %v to %v price %f, timesum %s",
			pi.Start.Round(time.Second), pi.End.Round(time.Second),
			pi.Price, sum)

		// current timeslot is a cheap one
		if pi.Start.Before(time.Now()) && pi.End.After(time.Now()) && duration > 0 {
			cheapSlotNow = true
			curSlotNr = cntExpectedSlots
			t.log.TRACE.Printf(" (now, slot number %v)", curSlotNr)
		}

		// we found all necessary cheap slots to charge to targetSoC
		if sum > duration {
			break
		}
	}

	if cheapSlotNow {
		// use the most expensive slot as little as possible, but do not disable on last charging slot
		if curSlotNr == cntExpectedSlots && !(t.cheapactive && cntExpectedSlots == 1) {
			if sum <= duration {
				t.log.DEBUG.Printf("cheap timeslot, charging...\n")
				t.cheapactive = true
			} else {
				if t.cheapactive && sum > duration+10*time.Minute {
					t.log.DEBUG.Printf("cheap timeslot, delayed start for %s\n", (sum - duration).Round(time.Minute))
					t.cheapactive = false
				}
			}
		} else { /* not most expensive slot */
			t.log.DEBUG.Printf("cheap timeslot, charging...\n")
			t.cheapactive = true
		}
	} else {
		t.log.DEBUG.Printf("not cheap, not charging...\n")
		t.cheapactive = false
	}

	return t.cheapactive, nil
}

func (t *Planner) IsCheap(duration time.Duration, end time.Time) (bool, error) {
	t.mux.Lock()
	defer t.mux.Unlock()

	if end.Before(time.Now()) {
		t.cheapactive = false
		return false, nil
	}

	duration = time.Duration(float64(duration) * 1.05) // increase by 5%

	// Save same duration until next price info update
	if end.After(t.last) {
		duration_old := duration
		duration = time.Duration(float64(duration) * float64(time.Until(t.last)) / float64(time.Until(end)))
		t.log.DEBUG.Printf("reduced duration from %s to %s until got new priceinfo after %s\n", duration_old.Round(time.Minute), duration.Round(time.Minute), t.last.Round(time.Minute))
	}

	t.log.DEBUG.Printf("charge duration: %s, end: %v, find best prices:\n", duration.Round(time.Minute), end.Round(time.Second))

	cheapactive, err := t.isCheapSlotNow(duration, end)
	if err != nil {
		return false, err
	}

	if cheapactive {
		return true, nil
	}

	isCheap, err := t.tariff.IsCheap()
	if isCheap && err == nil {
		t.log.DEBUG.Printf("low marketprice, charging")
	}

	return isCheap, err
}
