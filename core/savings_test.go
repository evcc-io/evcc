package core

import (
	"testing"
	"time"
)

func mockTime(t string) time.Time {
	time, _ := time.Parse("2006-01-02 15:04:05", "2021-10-04 "+t)
	return time
}

func assert(t *testing.T, s Savings, total int, self int, percentage float64) {
	if s.ChargedTotal() != total {
		t.Errorf("ChargedTotal was incorrect, got: %d, want: %d.", s.ChargedTotal(), total)
	}
	if s.ChargedSelfConsumption() != self {
		t.Errorf("ChargedSelfConsumption was incorrect, got: %d, want: %d.", s.ChargedSelfConsumption(), self)
	}
	if int(s.SelfPercentage()) != int(percentage) {
		t.Errorf("SelfPercentage was incorrect, got: %.1f, want: %.1f.", s.SelfPercentage(), percentage)
	}
}

func TestSavingsWithChangingEnergySources(t *testing.T) {
	s := *NewSavings()

	s.Update(0, 0, 0, 0, mockTime("12:00:00"))

	// 1 hour charge, half grid, half pv
	s.Update(2500, 2500, 0, 5000, mockTime("13:00:00"))
	assert(t, s, 5000, 2500, 50)

	// 1 hour charge, full pv
	s.Update(0, 5000, 0, 5000, mockTime("14:00:00"))
	assert(t, s, 10000, 7500, 75)

	// 1 hour charge, full grid
	s.Update(5000, 0, 0, 5000, mockTime("15:00:00"))
	assert(t, s, 15000, 7500, 50)

	// 1 hour charge, half grid, half battery
	s.Update(2500, 0, 2500, 5000, mockTime("16:00:00"))
	assert(t, s, 20000, 10000, 50)

	// 1 hour charge, full pv, pv export
	s.Update(-5000, 10000, 0, 5000, mockTime("17:00:00"))
	assert(t, s, 25000, 15000, 60)

	// 1 hour charge, full pv, pv export, battery charge
	s.Update(-2500, 10000, -2500, 5000, mockTime("18:00:00"))
	assert(t, s, 30000, 20000, 66)

	// 1 hour charge, double charge speed, full grid
	s.Update(10000, 0, 0, 10000, mockTime("19:00:00"))
	assert(t, s, 40000, 20000, 50)
}

func TestSavingsWithDifferentTimespans(t *testing.T) {
	s := *NewSavings()

	s.Update(0, 0, 0, 0, mockTime("12:00:00"))

	// 10 second 11kW charging, full grid
	s.Update(11000, 0, 0, 11000, mockTime("12:00:10"))
	assert(t, s, 30, 0, 0) // 30,555Wh

	// 10 second 11kW charging, full grid
	s.Update(11000, 0, 0, 11000, mockTime("12:00:20"))
	assert(t, s, 61, 0, 0) // 61,111Wh

	// 5x 2 second 11kW charging, full grid
	s.Update(11000, 0, 0, 11000, mockTime("12:00:22"))
	s.Update(11000, 0, 0, 11000, mockTime("12:00:24"))
	s.Update(11000, 0, 0, 11000, mockTime("12:00:26"))
	s.Update(11000, 0, 0, 11000, mockTime("12:00:28"))
	s.Update(11000, 0, 0, 11000, mockTime("12:00:30"))
	assert(t, s, 91, 0, 0) // 91,666Wh

	// 30 min 11kW charging, full grid
	s.Update(11000, 0, 0, 11000, mockTime("12:30:20"))
	assert(t, s, 5561, 0, 0) // 5561,111Wh

	// 6 hours 11kW charging, full pv
	s.Update(0, 11000, 0, 11000, mockTime("16:30:20"))
	assert(t, s, 49561, 44000, 88)
}
