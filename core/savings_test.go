package core

import (
	"math"
	"testing"
	"time"
)

func mockTime(t string) time.Time {
	time, _ := time.Parse("2006-01-02 15:04:05", "2021-10-04 "+t)
	return time
}

func assert(t *testing.T, s Savings, total float64, self float64, percentage float64) {
	if !compareWithTolerane(s.ChargedTotal(), total) {
		t.Errorf("ChargedTotal was incorrect, got: %.3f, want: %.3f.", s.ChargedTotal(), total)
	}
	if !compareWithTolerane(s.ChargedSelfConsumption(), self) {
		t.Errorf("ChargedSelfConsumption was incorrect, got: %.3f, want: %.3f.", s.ChargedSelfConsumption(), self)
	}
	if int(s.SelfPercentage()) != int(percentage) {
		t.Errorf("SelfPercentage was incorrect, got: %.1f, want: %.1f.", s.SelfPercentage(), percentage)
	}
}

func compareWithTolerane(a, b float64) bool {
	tolerance := 0.001
	diff := math.Abs(a - b)
	return diff < tolerance
}

func TestSavingsWithChangingEnergySources(t *testing.T) {
	s := *NewSavings()

	s.Update(0, 0, 0, 0, mockTime("12:00:00"))

	// 1 hour charge, half grid, half pv
	s.Update(2500, 2500, 0, 5000, mockTime("13:00:00"))
	assert(t, s, 5, 2.5, 50)

	// 1 hour charge, full pv
	s.Update(0, 5000, 0, 5000, mockTime("14:00:00"))
	assert(t, s, 10, 7.5, 75)

	// 1 hour charge, full grid
	s.Update(5000, 0, 0, 5000, mockTime("15:00:00"))
	assert(t, s, 15, 7.5, 50)

	// 1 hour charge, half grid, half battery
	s.Update(2500, 0, 2500, 5000, mockTime("16:00:00"))
	assert(t, s, 20, 10, 50)

	// 1 hour charge, full pv, pv export
	s.Update(-5000, 10000, 0, 5000, mockTime("17:00:00"))
	assert(t, s, 25, 15, 60)

	// 1 hour charge, full pv, pv export, battery charge
	s.Update(-2500, 10000, -2500, 5000, mockTime("18:00:00"))
	assert(t, s, 30, 20, 66)

	// 1 hour charge, double charge speed, full grid
	s.Update(10000, 0, 0, 10000, mockTime("19:00:00"))
	assert(t, s, 40, 20, 50)
}

func TestSavingsWithDifferentTimespans(t *testing.T) {
	s := *NewSavings()

	s.Update(0, 0, 0, 0, mockTime("12:00:00"))

	// 10 second 11kW charging, full grid
	s.Update(11000, 0, 0, 11000, mockTime("12:00:10"))
	assert(t, s, 0.030556, 0, 0) // 30,555Wh

	// 10 second 11kW charging, full grid
	s.Update(11000, 0, 0, 11000, mockTime("12:00:20"))
	assert(t, s, 0.061111, 0, 0) // 61,111Wh

	// 5x 2 second 11kW charging, full grid
	s.Update(11000, 0, 0, 11000, mockTime("12:00:22"))
	s.Update(11000, 0, 0, 11000, mockTime("12:00:24"))
	s.Update(11000, 0, 0, 11000, mockTime("12:00:26"))
	s.Update(11000, 0, 0, 11000, mockTime("12:00:28"))
	s.Update(11000, 0, 0, 11000, mockTime("12:00:30"))
	assert(t, s, 0.092, 0, 0) // 91,666Wh

	// 30 min 11kW charging, full grid
	s.Update(11000, 0, 0, 11000, mockTime("12:30:20"))
	assert(t, s, 5.561, 0, 0) // 5561,111Wh

	// 6 hours 11kW charging, full pv
	s.Update(0, 11000, 0, 11000, mockTime("16:30:20"))
	assert(t, s, 49.561, 44, 88)
}
