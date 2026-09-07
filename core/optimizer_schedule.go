package core

import (
	"iter"
	"time"
)

// Result indices keep their original intervals even after earlier slots expire.
type optimizerSchedule struct {
	timestamps []time.Time
	dt         []int
}

func (s optimizerSchedule) len() int {
	return min(len(s.timestamps), len(s.dt))
}

func (s optimizerSchedule) duration(slot int) time.Duration {
	if slot < 0 || slot >= s.len() {
		return 0
	}
	return time.Duration(s.dt[slot]) * time.Second
}

func (s optimizerSchedule) end(slot int) time.Time {
	if slot < 0 || slot >= s.len() {
		return time.Time{}
	}
	return s.timestamps[slot].Add(s.duration(slot))
}

func (s optimizerSchedule) activeSlot(at time.Time) int {
	for i := range s.len() {
		if at.Before(s.end(i)) {
			return i
		}
	}
	return -1
}

// Keep control policy separate so it cannot shift forecast intervals.
func (s optimizerSchedule) controlSlot(at time.Time) int {
	return s.activeSlot(at)
}

func (s optimizerSchedule) remainingSlots(at time.Time) iter.Seq[int] {
	return func(yield func(int) bool) {
		for i := s.activeSlot(at); i >= 0 && i < s.len(); i++ {
			if !yield(i) {
				return
			}
		}
	}
}
