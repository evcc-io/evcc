package core

import (
	"math"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
)

func assert(t *testing.T, s *Savings, total float64, self float64, percentage float64) {
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
	clck := clock.NewMock()
	s := &Savings{
		log:     util.NewLogger("savings"),
		clock:   clck,
		started: clck.Now(),
		updated: clck.Now(),
	}

	s.Update(0, 0, 0, 0)

	// half grid, half pv
	clck.Add(time.Hour)
	s.Update(2500, 2500, 0, 5000)
	assert(t, s, 5, 2.5, 50)

	// full pv
	clck.Add(time.Hour)
	s.Update(0, 5000, 0, 5000)
	assert(t, s, 10, 7.5, 75)

	// full grid
	clck.Add(time.Hour)
	s.Update(5000, 0, 0, 5000)
	assert(t, s, 15, 7.5, 50)

	// half grid, half battery
	clck.Add(time.Hour)
	s.Update(2500, 0, 2500, 5000)
	assert(t, s, 20, 10, 50)

	// full pv, pv export
	clck.Add(time.Hour)
	s.Update(-5000, 10000, 0, 5000)
	assert(t, s, 25, 15, 60)

	// full pv, pv export, battery charge
	clck.Add(time.Hour)
	s.Update(-2500, 10000, -2500, 5000)
	assert(t, s, 30, 20, 66)

	// double charge speed, full grid
	clck.Add(time.Hour)
	s.Update(10000, 0, 0, 10000)
	assert(t, s, 40, 20, 50)
}

func TestSavingsWithDifferentTimespans(t *testing.T) {
	clck := clock.NewMock()
	s := &Savings{
		log:     util.NewLogger("savings"),
		started: clck.Now(),
		updated: clck.Now(),
		clock:   clck,
	}

	s.Update(0, 0, 0, 0)

	// 10 second 11kW charging, full grid
	clck.Add(10 * time.Second)
	s.Update(11000, 0, 0, 11000)
	assert(t, s, 0.030556, 0, 0) // 30,555Wh

	// 10 second 11kW charging, full grid
	clck.Add(10 * time.Second)
	s.Update(11000, 0, 0, 11000)
	assert(t, s, 0.061111, 0, 0) // 61,111Wh

	// 5x 2 second 11kW charging, full grid
	clck.Add(2 * time.Second)
	s.Update(11000, 0, 0, 11000)
	clck.Add(2 * time.Second)
	s.Update(11000, 0, 0, 11000)
	clck.Add(2 * time.Second)
	s.Update(11000, 0, 0, 11000)
	clck.Add(2 * time.Second)
	s.Update(11000, 0, 0, 11000)
	clck.Add(2 * time.Second)
	s.Update(11000, 0, 0, 11000)
	assert(t, s, 0.092, 0, 0) // 91,666Wh

	// 30 min 11kW charging, full grid
	clck.Add(30 * time.Minute)
	s.Update(11000, 0, 0, 11000)
	assert(t, s, 5.592, 0, 0) // 5561,111Wh

	// 4 hours 11kW charging, full pv
	clck.Add(4 * time.Hour)
	s.Update(0, 11000, 0, 11000)
	assert(t, s, 49.592, 44, 88)
}
