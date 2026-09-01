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

// TestUpdateStatusRefreshesConnectedWithoutControl verifies that UpdateStatus
// picks up a charger status change - and hence the connected/charging state
// shown in the UI - without executing any charging control logic.
//
// This is the path taken when site meters are unavailable: a failing meter must
// not freeze vehicle connection state (see #33395).
func TestUpdateStatusRefreshesConnectedWithoutControl(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock.NewMock(),
		charger:     charger,
		chargeMeter: &Null{},
		chargeRater: &Null{},
		chargeTimer: &Null{},
		wakeUpTimer: NewTimer(),
		minCurrent:  minA,
		maxCurrent:  maxA,
		phases:      1,
		status:      api.StatusA, // disconnected
	}

	attachListeners(t, lp)

	// vehicle plugs in between cycles
	charger.EXPECT().Status().Return(api.StatusB, nil)

	// Deliberately no Enable()/MaxCurrent() expectations: UpdateStatus must not
	// drive the charger. gomock fails the test if it does.
	lp.UpdateStatus()

	assert.Equal(t, api.StatusB, lp.GetStatus(), "status should be refreshed")
	assert.True(t, lp.connected(), "vehicle should be reported as connected")

	ctrl.Finish()
}

// TestUpdateStatusPropagatesChargerError verifies UpdateStatus tolerates a
// charger read failure without panicking and leaves status unchanged.
func TestUpdateStatusPropagatesChargerError(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock.NewMock(),
		charger:     charger,
		chargeMeter: &Null{},
		chargeRater: &Null{},
		chargeTimer: &Null{},
		wakeUpTimer: NewTimer(),
		minCurrent:  minA,
		maxCurrent:  maxA,
		phases:      1,
		status:      api.StatusB,
	}

	attachListeners(t, lp)

	charger.EXPECT().Status().Return(api.StatusNone, api.ErrTimeout)

	lp.UpdateStatus()

	assert.Equal(t, api.StatusB, lp.GetStatus(), "status should be unchanged on error")

	ctrl.Finish()
}
