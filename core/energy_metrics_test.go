package core

import (
	"testing"
)

func isEqualFloat64(a, b *float64) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}

func TestEnergyMetrics(t *testing.T) {
	f := func(f float64) *float64 { return &f }

	type tcStep = struct {
		kWh, greenShare  float64
		effPrice, effCo2 *float64
	}

	tc := []struct {
		title                         string
		steps                         []tcStep
		totalWh, solarPercentage      float64
		price, pricePerKWh, co2PerKWh *float64
	}{
		{"initial state",
			[]tcStep{},
			0, 0, nil, nil, nil,
		},
		{"energy value",
			[]tcStep{
				{0.1, 0, nil, nil},
				{0.2, 0, nil, nil},
			},
			200, 0, nil, nil, nil,
		},
		{"ignore lower energy value",
			[]tcStep{
				{0.2, 0, nil, nil},
				{0.1, 0, nil, nil},
			},
			200, 0, nil, nil, nil,
		},
		{"half solar",
			[]tcStep{
				{0.1, 1, nil, nil},
				{0.2, 0, nil, nil},
			},
			200, 50, nil, nil, nil,
		},
		{"only solar",
			[]tcStep{
				{0.1, 1, nil, nil},
				{0.2, 1, nil, nil},
			},
			200, 100, nil, nil, nil,
		},
		{"static price",
			[]tcStep{
				{1, 0, f(0.5), nil},
			},
			1000, 0, f(0.5), f(0.5), nil,
		},
		{"dynamic price",
			[]tcStep{
				{1, 0, f(1), nil},
				{2, 0, f(0), nil},
			},
			2000, 0, f(1), f(0.5), nil,
		},
		{"dynamic price",
			[]tcStep{
				{2, 0, f(1), nil},
				{4, 0, f(0), nil},
			},
			4000, 0, f(2), f(0.5), nil,
		},
		{"static co2",
			[]tcStep{
				{1, 0, nil, f(500)},
			},
			1000, 0, nil, nil, f(500),
		},
		{"dynamic co2",
			[]tcStep{
				{1, 0, nil, f(1000)},
				{2, 0, nil, f(0)},
			},
			2000, 0, nil, nil, f(500),
		},
		{"grid only, half, full solar, half, grid only",
			[]tcStep{
				{1, 0, f(2), f(200)},
				{2, 0.5, f(1), f(50)},
				{3, 1, f(0), f(0)},
				{4, 0.5, f(1), f(50)},
				{5, 0, f(2), f(200)},
			},
			5000, 40, f(6), f(1.2), f(100),
		},
	}

	for _, tc := range tc {
		//t.Logf("%+v", tc)

		s := NewEnergyMetrics()

		for _, tc := range tc.steps {
			s.SetEnvironment(tc.greenShare, tc.effPrice, tc.effCo2)
			s.Update(tc.kWh)
		}

		if s.TotalWh() != tc.totalWh {
			t.Errorf("%s: TotalWh was incorrect, got: %.3f, want: %.3f.", tc.title, s.TotalWh(), tc.totalWh)
		}
		if s.SolarPercentage() != tc.solarPercentage {
			t.Errorf("%s: SolarPercentage was incorrect, got: %.3f, want: %.3f.", tc.title, s.SolarPercentage(), tc.solarPercentage)
		}
		price := s.Price()
		if !isEqualFloat64(price, tc.price) {
			t.Errorf("%s: Price was incorrect, got: %v, want: %v.", tc.title, *price, *tc.price)
		}
		pricePerKWh := s.PricePerKWh()
		if !isEqualFloat64(pricePerKWh, tc.pricePerKWh) {
			t.Errorf("%s: PricePerKWh was incorrect, got: %v, want: %v.", tc.title, *pricePerKWh, *tc.pricePerKWh)
		}
		co2PerKWh := s.Co2PerKWh()
		if !isEqualFloat64(co2PerKWh, tc.co2PerKWh) {
			t.Errorf("%s: Co2PerKWh was incorrect, got: %v, want: %v.", tc.title, *co2PerKWh, *tc.co2PerKWh)
		}
	}

	// reset
	s := NewEnergyMetrics()
	s.SetEnvironment(1, f(1), f(1))
	s.Update(1)
	s.Reset()
	if s.TotalWh() != 0 || s.SolarPercentage() != 0 || s.Co2PerKWh() != nil || s.Price() != nil || s.PricePerKWh() != nil {
		t.Errorf("Metrics not properly reset %+v", s)
	}
}
