package prioritizer

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// testSettings is a static site-level priority configuration for tests
type testSettings struct {
	strategy   api.PriorityStrategy
	basis      api.PriorityBasis
	ref        float64
	hysteresis int
}

func (s *testSettings) GetPriorityStrategy() api.PriorityStrategy { return s.strategy }
func (s *testSettings) GetPriorityHysteresis() int                { return s.hysteresis }

func (s *testSettings) EffectivePriorityScoring() (api.PriorityBasis, float64) {
	if s.ref <= 0 {
		return s.basis, 100
	}
	return s.basis, s.ref
}

// mockLoadpoint returns a loadpoint mock with the given priority tier and score
func mockLoadpoint(ctrl *gomock.Controller, prio int, score float64) *loadpoint.MockAPI {
	return mockHeatingLoadpoint(ctrl, prio, score, false)
}

func mockHeatingLoadpoint(ctrl *gomock.Controller, prio int, score float64, heating bool) *loadpoint.MockAPI {
	lp := loadpoint.NewMockAPI(ctrl)
	lp.EXPECT().GetTitle().AnyTimes()
	lp.EXPECT().IsHeating().Return(heating).AnyTimes()
	lp.EXPECT().EffectivePriority().Return(prio).AnyTimes()
	lp.EXPECT().EffectivePriorityScore(gomock.Any(), gomock.Any(), gomock.Any()).Return(score).AnyTimes()
	return lp
}

func TestPrioritzer(t *testing.T) {
	ctrl := gomock.NewController(t)

	p := New(nil, &testSettings{})

	lo := mockLoadpoint(ctrl, 0, 0.0)
	hi := mockLoadpoint(ctrl, 1, 1.0)

	// no additional power available
	lo.EXPECT().GetChargePowerFlexibility(nil).Return(300.0)
	p.UpdateChargePowerFlexibility(lo, nil)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(lo))

	// additional power available
	hi.EXPECT().GetChargePowerFlexibility(nil).Return(1e3)
	p.UpdateChargePowerFlexibility(hi, nil)
	assert.Equal(t, 300.0, p.GetChargePowerFlexibility(hi))

	// additional power removed
	lo.EXPECT().GetChargePowerFlexibility(nil).Return(0.0)
	p.UpdateChargePowerFlexibility(lo, nil)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(hi))
}

// TestPrioritizerWithinTier verifies that loadpoints sharing the same priority
// tier are ranked by their fractional score (e.g. soc/deficit strategy), so the
// emptier vehicle takes surplus from the fuller one.
func TestPrioritizerWithinTier(t *testing.T) {
	ctrl := gomock.NewController(t)

	p := New(nil, &testSettings{strategy: api.PrioritySoc})

	full := mockLoadpoint(ctrl, 0, 0.20)  // soc 80
	empty := mockLoadpoint(ctrl, 0, 0.80) // soc 20

	// fuller vehicle has nothing below it -> no extra power
	full.EXPECT().GetChargePowerFlexibility(nil).Return(500.0)
	p.UpdateChargePowerFlexibility(full, nil)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(full))

	// emptier vehicle (higher score in the same tier) takes the fuller one's flexible power
	empty.EXPECT().GetChargePowerFlexibility(nil).Return(1e3)
	p.UpdateChargePowerFlexibility(empty, nil)
	assert.Equal(t, 500.0, p.GetChargePowerFlexibility(empty))
}

// TestPrioritizerHysteresis verifies the priority deadband: within the same tier,
// a loadpoint only outranks another when ahead by more than the configured band, so
// near-equal soc loadpoints tie (no stealing, no leapfrog) while clearly-emptier ones
// still take priority.
func TestPrioritizerHysteresis(t *testing.T) {
	ctrl := gomock.NewController(t)

	// 5% deadband (0.05)
	p := New(nil, &testSettings{strategy: api.PrioritySoc, hysteresis: 5})

	a := mockLoadpoint(ctrl, 0, 0.50) // soc 50
	b := mockLoadpoint(ctrl, 0, 0.49) // soc 51
	c := mockLoadpoint(ctrl, 0, 0.60) // soc 40, clearly emptier

	b.EXPECT().GetChargePowerFlexibility(nil).Return(400.0)
	p.UpdateChargePowerFlexibility(b, nil)

	// a is only 0.01 ahead of b -> within the 0.05 band -> no steal (no leapfrog)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(a))

	// c is 0.11 ahead of b -> beyond the band -> takes b's flexible power
	assert.Equal(t, 400.0, p.GetChargePowerFlexibility(c))
}

// TestPrioritizerHysteresisTierGate verifies that the deadband sub-orders within a
// tier only: an explicitly configured priority must win even when the scores are
// barely apart (2.00 vs 1.99).
func TestPrioritizerHysteresisTierGate(t *testing.T) {
	ctrl := gomock.NewController(t)

	p := New(nil, &testSettings{strategy: api.PrioritySoc, hysteresis: 5})

	hi := mockLoadpoint(ctrl, 2, 2.00) // prio 2, soc 100
	lo := mockLoadpoint(ctrl, 1, 1.99) // prio 1, soc 1

	lo.EXPECT().GetChargePowerFlexibility(nil).Return(400.0)
	p.UpdateChargePowerFlexibility(lo, nil)

	assert.Equal(t, 400.0, p.GetChargePowerFlexibility(hi))
}

// TestPrioritizerHeatingSameTier verifies that a same-tier pair involving heating is
// left untouched: heating aliases temperature as soc and carries no comparable score.
func TestPrioritizerHeatingSameTier(t *testing.T) {
	ctrl := gomock.NewController(t)

	p := New(nil, &testSettings{strategy: api.PrioritySoc})

	heater := mockHeatingLoadpoint(ctrl, 0, 0.0, true)
	car := mockLoadpoint(ctrl, 0, 0.40)

	heater.EXPECT().GetChargePowerFlexibility(nil).Return(800.0)
	p.UpdateChargePowerFlexibility(heater, nil)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(car), "car must not take the heater's power")

	car.EXPECT().GetChargePowerFlexibility(nil).Return(1e3)
	p.UpdateChargePowerFlexibility(car, nil)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(heater), "heater must not take the car's power")
}

// TestPrioritizerHysteresisEnergyUnit verifies that under the energy basis the
// hysteresis is a kWh band, normalised against the same reference as the score.
func TestPrioritizerHysteresisEnergyUnit(t *testing.T) {
	ctrl := gomock.NewController(t)

	// 10 kWh deadband against a 200 kWh reference -> 0.05
	p := New(nil, &testSettings{strategy: api.PrioritySoc, basis: api.PriorityBasisEnergy, ref: 200, hysteresis: 10})

	a := mockLoadpoint(ctrl, 0, 0.36) // 72 kWh gap
	b := mockLoadpoint(ctrl, 0, 0.30) // 60 kWh gap, 12 kWh behind a
	c := mockLoadpoint(ctrl, 0, 0.32) // 64 kWh gap, 8 kWh behind a

	b.EXPECT().GetChargePowerFlexibility(nil).Return(500.0)
	p.UpdateChargePowerFlexibility(b, nil)
	c.EXPECT().GetChargePowerFlexibility(nil).Return(300.0)
	p.UpdateChargePowerFlexibility(c, nil)

	// only b is beyond the 10 kWh band - as a percentage band it would be 10%/0.10 and neither would be
	assert.Equal(t, 500.0, p.GetChargePowerFlexibility(a))
}
