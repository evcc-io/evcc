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
	hysteresis int
}

func (s *testSettings) GetPriorityStrategy() api.PriorityStrategy { return s.strategy }
func (s *testSettings) GetPriorityBasis() api.PriorityBasis       { return s.basis }
func (s *testSettings) GetPriorityHysteresis() int                { return s.hysteresis }

func mockLoadpoint(ctrl *gomock.Controller, prio int, gap float64) *loadpoint.MockAPI {
	return mockGapLoadpoint(ctrl, prio, gap, true)
}

func mockGapLoadpoint(ctrl *gomock.Controller, prio int, gap float64, comparable bool) *loadpoint.MockAPI {
	lp := loadpoint.NewMockAPI(ctrl)
	lp.EXPECT().GetTitle().AnyTimes()
	lp.EXPECT().GetStatus().Return(api.StatusB).AnyTimes()
	lp.EXPECT().GetVehicle().Return(nil).AnyTimes()
	lp.EXPECT().EffectivePriority().Return(prio).AnyTimes()
	lp.EXPECT().PriorityGap(gomock.Any(), gomock.Any()).Return(gap, comparable).AnyTimes()
	return lp
}

type gapState struct {
	gap     float64
	ok      bool
	status  api.ChargeStatus
	vehicle api.Vehicle
}

func mutableLoadpoint(ctrl *gomock.Controller, state *gapState) *loadpoint.MockAPI {
	lp := loadpoint.NewMockAPI(ctrl)
	lp.EXPECT().GetTitle().AnyTimes()
	lp.EXPECT().EffectivePriority().Return(0).AnyTimes()
	lp.EXPECT().GetStatus().DoAndReturn(func() api.ChargeStatus { return state.status }).AnyTimes()
	lp.EXPECT().GetVehicle().DoAndReturn(func() api.Vehicle { return state.vehicle }).AnyTimes()
	lp.EXPECT().PriorityGap(gomock.Any(), gomock.Any()).DoAndReturn(func(api.PriorityStrategy, api.PriorityBasis) (float64, bool) {
		return state.gap, state.ok
	}).AnyTimes()
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

	a := mockLoadpoint(ctrl, 0, 50) // soc 50
	b := mockLoadpoint(ctrl, 0, 49) // soc 51
	c := mockLoadpoint(ctrl, 0, 60) // soc 40, clearly emptier

	b.EXPECT().GetChargePowerFlexibility(nil).Return(400.0)
	p.UpdateChargePowerFlexibility(b, nil)

	// a is only 0.01 ahead of b -> within the 0.05 band -> no steal (no leapfrog)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(a))

	// c is 0.11 ahead of b -> beyond the band -> takes b's flexible power
	assert.Equal(t, 400.0, p.GetChargePowerFlexibility(c))
}

func TestPrioritizerHysteresisLatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := New(nil, &testSettings{strategy: api.PrioritySoc, hysteresis: 3})

	stateA := &gapState{gap: 60, ok: true, status: api.StatusB}
	stateB := &gapState{gap: 50, ok: true, status: api.StatusB}
	a, b := mutableLoadpoint(ctrl, stateA), mutableLoadpoint(ctrl, stateB)
	assert.True(t, p.Outranks(a, b))
	assert.False(t, p.Outranks(b, a))

	stateA.gap, stateB.gap = 49, 50
	assert.True(t, p.Outranks(a, b), "winner holds inside the band")
	assert.False(t, p.Outranks(b, a))

	stateB.gap = 53.1
	assert.True(t, p.Outranks(b, a), "challenger takes over beyond the band")
	assert.False(t, p.Outranks(a, b))
}

func TestPrioritizerUnavailableGapClearsLatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := New(nil, &testSettings{strategy: api.PrioritySoc, hysteresis: 3})
	stateA := &gapState{gap: 60, ok: true, status: api.StatusB}
	stateB := &gapState{gap: 50, ok: true, status: api.StatusB}
	a, b := mutableLoadpoint(ctrl, stateA), mutableLoadpoint(ctrl, stateB)

	assert.True(t, p.Outranks(a, b))
	stateA.ok = false
	assert.False(t, p.Outranks(a, b))
	assert.False(t, p.Outranks(b, a))

	stateA.ok = true
	stateA.gap, stateB.gap = 51, 50
	assert.False(t, p.Outranks(a, b), "old lead must not revive")
	assert.False(t, p.Outranks(b, a))
}

func TestPrioritizerConfigChangeClearsLatches(t *testing.T) {
	changes := map[string]func(*testSettings){
		"strategy":   func(s *testSettings) { s.strategy = api.PriorityDeficit },
		"basis":      func(s *testSettings) { s.basis = api.PriorityBasisEnergy },
		"hysteresis": func(s *testSettings) { s.hysteresis = 4 },
	}

	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			settings := &testSettings{strategy: api.PrioritySoc, hysteresis: 3}
			p := New(nil, settings)
			stateA := &gapState{gap: 60, ok: true, status: api.StatusB}
			stateB := &gapState{gap: 50, ok: true, status: api.StatusB}
			a, b := mutableLoadpoint(ctrl, stateA), mutableLoadpoint(ctrl, stateB)

			assert.True(t, p.Outranks(a, b))
			stateA.gap, stateB.gap = 51, 50
			change(settings)
			assert.False(t, p.Outranks(a, b), "old lead must not survive configuration changes")
			assert.False(t, p.Outranks(b, a))
		})
	}
}

func TestPrioritizerVehicleChangeClearsLatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := New(nil, &testSettings{strategy: api.PrioritySoc, hysteresis: 3})
	first, second := api.NewMockVehicle(ctrl), api.NewMockVehicle(ctrl)
	stateA := &gapState{gap: 60, ok: true, status: api.StatusB, vehicle: first}
	stateB := &gapState{gap: 50, ok: true, status: api.StatusB}
	a, b := mutableLoadpoint(ctrl, stateA), mutableLoadpoint(ctrl, stateB)

	assert.True(t, p.Outranks(a, b))
	stateA.vehicle = second
	stateA.gap, stateB.gap = 51, 50
	assert.False(t, p.Outranks(a, b), "previous vehicle's lead must not transfer")
	assert.False(t, p.Outranks(b, a))
}

func TestPrioritizerDisconnectClearsLatch(t *testing.T) {
	ctrl := gomock.NewController(t)
	p := New(nil, &testSettings{strategy: api.PrioritySoc, hysteresis: 3})
	stateA := &gapState{gap: 60, ok: true, status: api.StatusB}
	stateB := &gapState{gap: 50, ok: true, status: api.StatusB}
	a, b := mutableLoadpoint(ctrl, stateA), mutableLoadpoint(ctrl, stateB)

	assert.True(t, p.Outranks(a, b))
	stateA.status = api.StatusA
	a.EXPECT().GetChargePowerFlexibility(nil).Return(0.0)
	p.UpdateChargePowerFlexibility(a, nil)

	stateA.status = api.StatusB
	stateA.gap, stateB.gap = 51, 50
	assert.False(t, p.Outranks(a, b), "lead must not survive a disconnect")
	assert.False(t, p.Outranks(b, a))
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

func TestPrioritizerUnknownSocTies(t *testing.T) {
	ctrl := gomock.NewController(t)

	p := New(nil, &testSettings{strategy: api.PrioritySoc})

	unknown := mockGapLoadpoint(ctrl, 0, 0, false)
	car := mockLoadpoint(ctrl, 0, 0.40)

	unknown.EXPECT().GetChargePowerFlexibility(nil).Return(800.0)
	p.UpdateChargePowerFlexibility(unknown, nil)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(car))

	car.EXPECT().GetChargePowerFlexibility(nil).Return(1e3)
	p.UpdateChargePowerFlexibility(car, nil)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(unknown))
}

// TestPrioritizerHysteresisEnergyUnit verifies that under the energy basis the
// hysteresis is a kWh band, normalised against the same reference as the score.
func TestPrioritizerHysteresisEnergyUnit(t *testing.T) {
	ctrl := gomock.NewController(t)

	p := New(nil, &testSettings{strategy: api.PrioritySoc, basis: api.PriorityBasisEnergy, hysteresis: 10})

	a := mockLoadpoint(ctrl, 0, 72) // 72 kWh gap
	b := mockLoadpoint(ctrl, 0, 60) // 60 kWh gap, 12 kWh behind a
	c := mockLoadpoint(ctrl, 0, 64) // 64 kWh gap, 8 kWh behind a

	b.EXPECT().GetChargePowerFlexibility(nil).Return(500.0)
	p.UpdateChargePowerFlexibility(b, nil)
	c.EXPECT().GetChargePowerFlexibility(nil).Return(300.0)
	p.UpdateChargePowerFlexibility(c, nil)

	// only b is beyond the 10 kWh band - as a percentage band it would be 10%/0.10 and neither would be
	assert.Equal(t, 500.0, p.GetChargePowerFlexibility(a))
}
