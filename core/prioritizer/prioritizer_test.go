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
	lo.EXPECT().GetChargePower().Return(300.0)
	p.Consume(lo)
	assert.Equal(t, 0.0, p.Consumable(lo))

	//  additional power available
	hi.EXPECT().GetChargePower().Return(1e3)
	p.Consume(hi)
	assert.Equal(t, 300.0, p.Consumable(hi))
}
