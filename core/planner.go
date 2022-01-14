package core

import (
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

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

	var pi api.Rate
	var sum time.Duration
	cheap_slot := false
	cnt_expected_slots := 0
	cur_slot_nr := 0

	data, err := t.tariff.Rates()
	if err != nil {
		return false, err
	}

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

		// timeslot already startet
		pstart := pi.Start
		if pstart.Before(time.Now()) {
			pstart = time.Now()
		}

		// timeslot ends after charge finish time
		pend := pi.End
		if pend.After(end) {
			pend = end
		}

		cnt_expected_slots++
		delta := pend.Sub(pstart)
		sum += delta
		t.log.TRACE.Printf("  Slot from: %v to %v price %f, timesum %s",
			pi.Start.Round(time.Second), pi.End.Round(time.Second),
			pi.Price, sum)

		// current timeslot is a cheap one
		if pi.Start.Before(time.Now()) && pi.End.After(time.Now()) && duration > 0 {
			cheap_slot = true // rename to cheapSlotNow
			cur_slot_nr = cnt_expected_slots
			t.log.TRACE.Printf(" (now, slot number %v)", cur_slot_nr)
		}

		// we found all necessary cheap slots to charge to targetSoC
		if sum > duration {
			break
		}
	}

	if cheap_slot {
		// use the most expensive slot as little as possible, but do not disable on last charging slot
		if cur_slot_nr == cnt_expected_slots && !(t.cheapactive && cnt_expected_slots == 1) {
			if sum <= duration {
				t.log.DEBUG.Printf("cheap timeslot, charging...\n")
				t.cheapactive = true
			} else {
				if t.cheapactive && sum > duration+10*time.Minute {
					t.log.DEBUG.Printf("cheap timeslot, delayed start for %s\n", (sum - duration).Round(time.Minute))
					t.cheapactive = false
				}
			}
		} else { /* not most expensiv slot */
			t.log.DEBUG.Printf("cheap timeslot, charging...\n")
			t.cheapactive = true
		}
	} else {
		t.log.DEBUG.Printf("not cheap, not charging...\n")
		t.cheapactive = false
	}

	if t.cheapactive {
		return true, nil
	}

	isCheap := pi.Price <= t.cheap // convert EUR/MWh to EUR/KWh
	if isCheap {
		t.log.DEBUG.Printf("low marketprice, charging")
	}

	return isCheap, nil
}
