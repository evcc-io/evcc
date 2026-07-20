package core

import (
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestBatteryBoostDefault(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp := &Loadpoint{
		log:               util.NewLogger("foo"),
		bus:               evbus.New(),
		clock:             clock.NewMock(),
		settings:          settings.NewDatabaseSettingsAdapter("test-battery-boost-default"),
		charger:           charger,
		chargeMeter:       &Null{},
		chargeRater:       &Null{},
		chargeTimer:       &Null{},
		wakeUpTimer:       NewTimer(),
		minCurrent:        minA,
		maxCurrent:        maxA,
		status:            api.StatusC,
		batteryBoostLimit: 100,
	}
	attachListeners(t, lp)

	// opt-in default is off
	assert.False(t, lp.GetBatteryBoostDefault(), "default should be off initially")

	// setter toggles the in-memory state
	lp.SetBatteryBoostDefault(true)
	assert.True(t, lp.GetBatteryBoostDefault(), "should be enabled after set")

	lp.SetBatteryBoostDefault(false)
	assert.False(t, lp.GetBatteryBoostDefault(), "should be disabled after unset")

	// battery boost (which the default re-arms on connect) is only available in PV modes
	lp.mode = api.ModeOff
	assert.Error(t, lp.SetBatteryBoost(true), "boost must be rejected outside PV modes")

	lp.mode = api.ModePV
	assert.NoError(t, lp.SetBatteryBoost(true), "boost must be accepted in PV mode")
	assert.NotEqual(t, boostDisabled, lp.GetBatteryBoost(), "boost should be active after enabling in PV mode")
}
