package prioritizer

import (
	"testing"

	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPrioritzer(t *testing.T) {
	ctrl := gomock.NewController(t)

	p := New(nil)

	lo := loadpoint.NewMockAPI(ctrl)
	lo.EXPECT().Title().AnyTimes()
	lo.EXPECT().GetPriority().Return(0).AnyTimes()       // prio 0
	lo.EXPECT().EffectivePriority().Return(0).AnyTimes() // prio 0

	hi := loadpoint.NewMockAPI(ctrl)
	hi.EXPECT().Title().AnyTimes()
	hi.EXPECT().GetPriority().Return(1).AnyTimes()       // prio 1
	hi.EXPECT().EffectivePriority().Return(1).AnyTimes() // prio 1

	// no additional power available
	lo.EXPECT().GetChargePowerFlexibility().Return(300.0)
	p.UpdateChargePowerFlexibility(lo)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(lo))

	// additional power available
	hi.EXPECT().GetChargePowerFlexibility().Return(1e3)
	p.UpdateChargePowerFlexibility(hi)
	assert.Equal(t, 300.0, p.GetChargePowerFlexibility(hi))

	// additional power removed
	lo.EXPECT().GetChargePowerFlexibility().Return(0.0)
	p.UpdateChargePowerFlexibility(lo)
	assert.Equal(t, 0.0, p.GetChargePowerFlexibility(hi))
}
