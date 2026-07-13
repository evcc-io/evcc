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

func TestPrioritzer(t *testing.T) {
	ctrl := gomock.NewController(t)

	p := New(nil, &testSettings{})

	lo := loadpoint.NewMockAPI(ctrl)
	lo.EXPECT().GetTitle().AnyTimes()
	lo.EXPECT().EffectivePriorityScore(gomock.Any(), gomock.Any()).Return(0.0).AnyTimes() // prio 0

	hi := loadpoint.NewMockAPI(ctrl)
	hi.EXPECT().GetTitle().AnyTimes()
	hi.EXPECT().EffectivePriorityScore(gomock.Any(), gomock.Any()).Return(1.0).AnyTimes() // prio 1

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

	full := loadpoint.NewMockAPI(ctrl) // prio 0, soc 80 -> score 0.20
	full.EXPECT().GetTitle().AnyTimes()
	full.EXPECT().EffectivePriorityScore(gomock.Any(), gomock.Any()).Return(0.20).AnyTimes()

	empty := loadpoint.NewMockAPI(ctrl) // prio 0, soc 20 -> score 0.80
	empty.EXPECT().GetTitle().AnyTimes()
	empty.EXPECT().EffectivePriorityScore(gomock.Any(), gomock.Any()).Return(0.80).AnyTimes()

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

	// soc 50 -> 0.50, soc 51 -> 0.49
	a := loadpoint.NewMockAPI(ctrl)
	a.EXPECT().GetTitle().AnyTimes()
	a.EXPECT().EffectivePriorityScore(gomock.Any(), gomock.Any()).Return(0.50).AnyTimes()

	b := loadpoint.NewMockAPI(ctrl)
	b.EXPECT().GetTitle().AnyTimes()
	b.EXPECT().EffectivePriorityScore(gomock.Any(), gomock.Any()).Return(0.49).AnyTimes()

	// clearly emptier (soc 40 -> 0.60)
	c := loadpoint.NewMockAPI(ctrl)
	c.EXPECT().GetTitle().AnyTimes()
	c.EXPECT().EffectivePriorityScore(gomock.Any(), gomock.Any()).Return(0.60).AnyTimes()

	b.EXPECT().GetChargePowerFlexibility(nil).Return(400.0)
	p.UpdateChargePowerFlexibility(b, nil)

	// a is only 0.01 ahead of b -> within the 0.05 band -> no steal (no leapfrog)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(a))

	// c is 0.11 ahead of b -> beyond the band -> takes b's flexible power
	assert.Equal(t, 400.0, p.GetChargePowerFlexibility(c))
}

// TestPrioritizerEnergyBasisMixedCapacity verifies that when one loadpoint has a
// known vehicle capacity and another does not under the site-level energy basis,
// the whole comparison set is ranked by percent rather than mixing a kWh fraction
// against a percentage fraction. Without the fallback the unconfigured vehicle's
// percentage gap would out-score the configured vehicle's (smaller) kWh gap and
// wrongly steal surplus.
//
//	known:   soc 20%, 50 kWh -> energy 0.40, percent 0.80
//	unknown: soc 50%, no cap  -> energy would fall back to percent 0.50
//
// With percent ranking the genuinely emptier "known" car (0.80) outranks "unknown"
// (0.50); the buggy mixed ranking would have unknown (0.50) beat known (0.40).
func TestPrioritizerEnergyBasisMixedCapacity(t *testing.T) {
	ctrl := gomock.NewController(t)

	p := New(nil, &testSettings{strategy: api.PrioritySoc, basis: api.PriorityBasisEnergy})

	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(50.0).AnyTimes()

	// known capacity, lower soc -> percent score 0.80 (energy 0.40)
	known := loadpoint.NewMockAPI(ctrl)
	known.EXPECT().GetTitle().AnyTimes()
	known.EXPECT().GetVehicle().Return(vehicle).AnyTimes()
	known.EXPECT().EffectivePriorityScore(api.PrioritySoc, api.PriorityBasisPercent).Return(0.80).AnyTimes()

	// unknown capacity, higher soc -> percent score 0.50
	unknown := loadpoint.NewMockAPI(ctrl)
	unknown.EXPECT().GetTitle().AnyTimes()
	unknown.EXPECT().GetVehicle().Return(nil).AnyTimes()
	unknown.EXPECT().EffectivePriorityScore(api.PrioritySoc, api.PriorityBasisPercent).Return(0.50).AnyTimes()

	// unknown (fuller, 0.50) has nothing emptier below it -> no extra power
	unknown.EXPECT().GetChargePowerFlexibility(nil).Return(700.0)
	p.UpdateChargePowerFlexibility(unknown, nil)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(unknown))

	// known (emptier, 0.80) outranks unknown and takes its flexible power
	known.EXPECT().GetChargePowerFlexibility(nil).Return(1e3)
	p.UpdateChargePowerFlexibility(known, nil)
	assert.Equal(t, 700.0, p.GetChargePowerFlexibility(known))
}

// TestPrioritizerEnergyBasisAllKnown verifies that the energy basis is actually
// used when every participating loadpoint has a known vehicle capacity.
func TestPrioritizerEnergyBasisAllKnown(t *testing.T) {
	ctrl := gomock.NewController(t)

	p := New(nil, &testSettings{strategy: api.PrioritySoc, basis: api.PriorityBasisEnergy})

	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(50.0).AnyTimes()

	// big gap in kWh -> energy score 0.40
	big := loadpoint.NewMockAPI(ctrl)
	big.EXPECT().GetTitle().AnyTimes()
	big.EXPECT().GetVehicle().Return(vehicle).AnyTimes()
	big.EXPECT().EffectivePriorityScore(api.PrioritySoc, api.PriorityBasisEnergy).Return(0.40).AnyTimes()

	// small gap in kWh -> energy score 0.15
	small := loadpoint.NewMockAPI(ctrl)
	small.EXPECT().GetTitle().AnyTimes()
	small.EXPECT().GetVehicle().Return(vehicle).AnyTimes()
	small.EXPECT().EffectivePriorityScore(api.PrioritySoc, api.PriorityBasisEnergy).Return(0.15).AnyTimes()

	small.EXPECT().GetChargePowerFlexibility(nil).Return(600.0)
	p.UpdateChargePowerFlexibility(small, nil)

	// larger energy deficit outranks the smaller one on the energy scale
	assert.Equal(t, 600.0, p.GetChargePowerFlexibility(big))
}
