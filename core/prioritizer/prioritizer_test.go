package prioritizer

import (
	"testing"

	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPrioritzer(t *testing.T) {
	ctrl := gomock.NewController(t)

	p := New()

	lo := loadpoint.NewMockAPI(ctrl)
	lo.EXPECT().Priority().Return(0).AnyTimes()

	hi := loadpoint.NewMockAPI(ctrl)
	hi.EXPECT().Priority().Return(1).AnyTimes()

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
