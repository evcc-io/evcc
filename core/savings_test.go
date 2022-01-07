package core

import (
	"math"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/util"
)

func assert(t *testing.T, s *Savings, total, self, percentage float64) {
	if !compareWithTolerane(s.ChargedTotal(), total) {
		t.Errorf("ChargedTotal was incorrect, got: %.3f, want: %.3f.", s.ChargedTotal(), total)
	}
	if !compareWithTolerane(s.chargedSelfConsumption, self) {
		t.Errorf("ChargedSelfConsumption was incorrect, got: %.3f, want: %.3f.", s.chargedSelfConsumption, self)
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

type StubPublisher struct{}

func (p StubPublisher) publish(key string, val interface{}) {}

func TestSavingsWithChangingEnergySources(t *testing.T) {
	p := StubPublisher{}

	clck := clock.NewMock()
	s := &Savings{
		log:     util.NewLogger("foo"),
		clock:   clck,
		started: clck.Now(),
		updated: clck.Now(),
	}

	tc := []struct {
		title                     string
		grid, pv, battery, charge float64
		total, self, percentage   float64
	}{
		{"half grid, half pv",
			2500, 2500, 0, 5000,
			5, 2.5, 50},
		{"full pv",
			0, 5000, 0, 5000,
			10, 7.5, 75},
		{"full grid",
			5000, 0, 0, 5000,
			15, 7.5, 50},
		{"half grid, half battery",
			2500, 0, 2500, 5000,
			20, 10, 50},
		{"full pv, pv export",
			-5000, 10000, 0, 5000,
			25, 15, 60},
		{"full pv, pv export, battery charge",
			-2500, 10000, -2500, 5000,
			30, 20, 66},
		{"double charge speed, full grid",
			10000, 0, 0, 10000,
			40, 20, 50},
	}

	s.Update(p, 0, 0, 0, 0)

	for _, tc := range tc {
		t.Logf("%+v", tc)

		clck.Add(time.Hour)
		s.Update(p, tc.grid, tc.pv, tc.battery, tc.charge)
		assert(t, s, tc.total, tc.self, tc.percentage)
	}
}

func TestSavingsWithDifferentTimespans(t *testing.T) {
	p := StubPublisher{}

	clck := clock.NewMock()
	s := &Savings{
		log:     util.NewLogger("foo"),
		clock:   clck,
		started: clck.Now(),
		updated: clck.Now(),
	}

	type tcStep = struct {
		dt                        time.Duration
		grid, pv, battery, charge float64
	}

	tc := []struct {
		title                   string
		steps                   []tcStep
		total, self, percentage float64
	}{
		{"10 second 11kW charging, full grid",
			[]tcStep{
				{10 * time.Second, 0, 0, 0, 11000},
			},
			0.030556, 0, 0, // 30,555Wh
		},
		{"10 second 11kW charging, full grid",
			[]tcStep{
				{10 * time.Second, 0, 0, 0, 11000},
			},
			0.061111, 0, 0, // 61,111Wh
		},
		{"5x 2 second 11kW charging, full grid",
			[]tcStep{
				{2 * time.Second, 0, 0, 0, 11000},
				{2 * time.Second, 0, 0, 0, 11000},
				{2 * time.Second, 0, 0, 0, 11000},
				{2 * time.Second, 0, 0, 0, 11000},
				{2 * time.Second, 0, 0, 0, 11000},
			},
			0.092, 0, 0, // 91,666Wh
		},
		{"30 min 11kW charging, full grid",
			[]tcStep{
				{30 * time.Minute, 0, 0, 0, 11000},
			},
			5.592, 0, 0, // 5561,111Wh
		},
		{"4 hours 11kW charging, full pv",
			[]tcStep{
				{4 * time.Hour, 0, 11000, 0, 11000},
			},
			49.592, 44, 88,
		},
	}

	s.Update(p, 0, 0, 0, 0)

	for _, tc := range tc {
		t.Logf("%+v", tc)

		for _, tc := range tc.steps {
			clck.Add(tc.dt)
			s.Update(p, tc.grid, tc.pv, tc.battery, tc.charge)
		}

		assert(t, s, tc.total, tc.self, tc.percentage)
	}
}
