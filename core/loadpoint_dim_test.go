package core

import (
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestLoadpointDim(t *testing.T) {
	ctrl := gomock.NewController(t)

	dimmer := api.NewMockDimmer(ctrl)
	charger := &struct {
		*api.MockCharger
		api.Dimmer
	}{api.NewMockCharger(ctrl), dimmer}
	charger.MockCharger.EXPECT().Enabled().Return(false, nil).AnyTimes()

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock.NewMock(),
		charger:     charger,
		chargeMeter: newChargeMeter(&Null{}),
		chargeRater: &Null{},
		chargeTimer: &Null{},
		wakeUpTimer: NewTimer(),
		minCurrent:  minA,
		maxCurrent:  maxA,
		phases:      1,
	}
	attachListeners(t, lp)

	dimmer.EXPECT().DimPower().Return(4200.0).AnyTimes()

	// dim with residual power
	dimmer.EXPECT().Dimmed().Return(false, nil)
	dimmer.EXPECT().Dim(true, 4200.0).Return(nil)
	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, &dimState{active: true})

	// unchanged decision is not re-applied
	dimmer.EXPECT().Dimmed().Return(true, nil)
	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, &dimState{active: true})

	// fallback while dimmed re-applies 0W
	dimmer.EXPECT().Dimmed().Return(true, nil)
	dimmer.EXPECT().Dim(true, 0.0).Return(nil)
	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, &dimState{active: true, fallback: true})

	assert.True(t, ctrl.Satisfied())
}
