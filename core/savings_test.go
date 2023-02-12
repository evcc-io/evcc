package core

import (
	"math"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
)

func assertEnergy(t *testing.T, s *Savings, total, self, percentage float64) {
	if !compareWithTolerane(s.TotalCharged(), total) {
		t.Errorf("TotalCharged was incorrect, got: %.3f, want: %.3f.", s.TotalCharged(), total)
	}
	if !compareWithTolerane(s.selfConsumptionCharged, self) {
		t.Errorf("ChargedSelfConsumption was incorrect, got: %.3f, want: %.3f.", s.selfConsumptionCharged, self)
	}
	if int(s.SelfConsumptionPercent()) != int(percentage) {
		t.Errorf("SelfConsumptionPercent was incorrect, got: %.1f, want: %.1f.", s.SelfConsumptionPercent(), percentage)
	}
}

func assertPrices(t *testing.T, s *Savings, effectivePrice, savingsAmount float64) {
	if !compareWithTolerane(s.EffectivePrice(), effectivePrice) {
		t.Errorf("EffectivePrice was incorrect, got: %.3f, want: %.3f.", s.EffectivePrice(), effectivePrice)
	}
	if !compareWithTolerane(s.SavingsAmount(), savingsAmount) {
		t.Errorf("SavingsAmount was incorrect, got: %.3f, want: %.3f.", s.SavingsAmount(), savingsAmount)
	}
}

func compareWithTolerane(a, b float64) bool {
	tolerance := 0.001
	diff := math.Abs(a - b)
	return diff < tolerance
}

type StubPublisher struct{}

func (p StubPublisher) publish(key string, val interface{}) {}

func TestSavingsWithDifferentTimespans(t *testing.T) {
	p := StubPublisher{}

	clck := clock.NewMock()
	s := &Savings{
		clock:   clck,
		started: clck.Now(),
		updated: clck.Now(),
	}

	type tcStep = struct {
		dt                 time.Duration
		greenShare, charge float64
	}

	tc := []struct {
		title                   string
		steps                   []tcStep
		total, self, percentage float64
	}{
		{"10 second not charging, full grid",
			[]tcStep{
				{10 * time.Second, 0, 0},
			},
			0, 0, 0, // 0Wh
		},
		{"10 second 11kW charging, full grid",
			[]tcStep{
				{10 * time.Second, 0, 11000},
			},
			0.030556, 0, 0, // 30,555Wh
		},
		{"10 second 11kW charging, full grid",
			[]tcStep{
				{10 * time.Second, 0, 11000},
			},
			0.061111, 0, 0, // 61,111Wh
		},
		{"5x 2 second 11kW charging, full grid",
			[]tcStep{
				{2 * time.Second, 0, 11000},
				{2 * time.Second, 0, 11000},
				{2 * time.Second, 0, 11000},
				{2 * time.Second, 0, 11000},
				{2 * time.Second, 0, 11000},
			},
			0.092, 0, 0, // 91,666Wh
		},
		{"30 min 11kW charging, full grid",
			[]tcStep{
				{30 * time.Minute, 0, 11000},
			},
			5.592, 0, 0, // 5561,111Wh
		},
		{"4 hours 11kW charging, full pv",
			[]tcStep{
				{4 * time.Hour, 1, 11000},
			},
			49.592, 44, 88,
		},
	}

	s.Update(p, 0, 0)

	for _, tc := range tc {
		t.Logf("%+v", tc)

		for _, tc := range tc.steps {
			clck.Add(tc.dt)
			s.Update(p, tc.greenShare, tc.charge)
		}

		assertEnergy(t, s, tc.total, tc.self, tc.percentage)
	}
}

func TestEffectiveEnergyPriceAndSavingsAmount(t *testing.T) {
	p := StubPublisher{}

	clck := clock.NewMock()

	type tcStep = struct {
		dt                 time.Duration
		greenShare, charge float64
	}

	tc := []struct {
		title                         string
		steps                         []tcStep
		effectivePrice, savingsAmount float64
	}{
		{"1 hour, 10kW, full grid",
			[]tcStep{
				{time.Hour, 0, 10000},
			},
			0.3, 0,
		},
		{"1 hour, 10kW, full pv",
			[]tcStep{
				{time.Hour, 1, 10000},
			},
			0.08, 2.2,
		},
		{"1 hour, 10kW, full battery",
			[]tcStep{
				{time.Hour, 1, 10000},
			},
			0.08, 2.2,
		},

		{"1 hour, 10kW, half grid, half pv",
			[]tcStep{
				{time.Hour, 0.5, 10000},
			},
			0.19, 1.1,
		},
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)

		s := &Savings{
			clock:   clck,
			started: clck.Now(),
			updated: clck.Now(),
		}
		s.Update(p, 0, 0)

		for _, tc := range tc.steps {
			clck.Add(tc.dt)
			s.Update(p, tc.greenShare, tc.charge)
		}

		assertPrices(t, s, tc.effectivePrice, tc.savingsAmount)
	}
}
