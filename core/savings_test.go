package core

import (
	"testing"
	"time"
)

func mockTime(t string) time.Time {
	time, _ := time.Parse("2006-01-02 15:04", "2021-10-04 "+t)
	return time
}

func assert(t *testing.T, s Savings, total int64, self int64, percentage int64) {
	if s.ChargedTotal() != total {
		t.Errorf("ChargedTotal was incorrect, got: %d, want: %d.", s.ChargedTotal(), total)
	}
	if s.ChargedSelfConsumption() != self {
		t.Errorf("ChargedSelfConsumption was incorrect, got: %d, want: %d.", s.ChargedSelfConsumption(), self)
	}
	if int64(s.SelfPercentage()) != percentage {
		t.Errorf("ChargedTotal was incorrect, got: %d, want: %d.", int(s.SelfPercentage()), percentage)
	}
}

func TestSavings(t *testing.T) {
	s := *NewSavings()

	s.Update(0, 0, 0, 0, mockTime("12:00"))

	// 1 hour charge, half grid, half pv
	s.Update(2500, 2500, 0, 5000, mockTime("13:00"))
	assert(t, s, 5000, 2500, 50)

	// 1 hour charge, full pv
	s.Update(0, 5000, 0, 5000, mockTime("14:00"))
	assert(t, s, 10000, 7500, 75)

	// 1 hour charge, full grid
	s.Update(5000, 0, 0, 5000, mockTime("15:00"))
	assert(t, s, 15000, 7500, 50)

	// 1 hour charge, half grid, half battery
	s.Update(2500, 0, 2500, 5000, mockTime("16:00"))
	assert(t, s, 20000, 10000, 50)

	// 1 hour charge, full pv, pv export
	s.Update(-5000, 10000, 0, 5000, mockTime("17:00"))
	assert(t, s, 25000, 15000, 60)

	// 1 hour charge, full pv, pv export, battery charge
	s.Update(-2500, 10000, -2500, 5000, mockTime("18:00"))
	assert(t, s, 30000, 20000, 66)

	// 1 hour charge, double charge speed, full grid
	s.Update(10000, 0, 0, 10000, mockTime("19:00"))
	assert(t, s, 40000, 20000, 50)
}
