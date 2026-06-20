package prioritizer

import (
	"testing"

	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestPrioritzer(t *testing.T) {
	ctrl := gomock.NewController(t)

	p := New(nil)

	lo := loadpoint.NewMockAPI(ctrl)
	lo.EXPECT().GetTitle().AnyTimes()
	lo.EXPECT().EffectivePriorityScore().Return(0.0).AnyTimes() // prio 0

	hi := loadpoint.NewMockAPI(ctrl)
	hi.EXPECT().GetTitle().AnyTimes()
	hi.EXPECT().EffectivePriorityScore().Return(1.0).AnyTimes() // prio 1

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

	p := New(nil)

	full := loadpoint.NewMockAPI(ctrl) // prio 0, soc 80 -> score 0.20
	full.EXPECT().GetTitle().AnyTimes()
	full.EXPECT().EffectivePriorityScore().Return(0.20).AnyTimes()

	empty := loadpoint.NewMockAPI(ctrl) // prio 0, soc 20 -> score 0.80
	empty.EXPECT().GetTitle().AnyTimes()
	empty.EXPECT().EffectivePriorityScore().Return(0.80).AnyTimes()

	// fuller vehicle has nothing below it -> no extra power
	full.EXPECT().GetChargePowerFlexibility(nil).Return(500.0)
	p.UpdateChargePowerFlexibility(full, nil)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(full))

	// emptier vehicle (higher score in the same tier) takes the fuller one's flexible power
	empty.EXPECT().GetChargePowerFlexibility(nil).Return(1e3)
	p.UpdateChargePowerFlexibility(empty, nil)
	assert.Equal(t, 500.0, p.GetChargePowerFlexibility(empty))
}
