package core

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestStatusEvents(t *testing.T) {
	tc := []struct {
		from, to api.ChargeStatus
		events   []string
	}{
		{api.StatusNone, api.StatusA, []string{evVehicleDisconnect}},
		{api.StatusNone, api.StatusB, []string{evVehicleConnect}},
		{api.StatusNone, api.StatusC, []string{evVehicleConnect, evChargeStart}},

		{api.StatusA, api.StatusB, []string{evVehicleConnect}},
		{api.StatusA, api.StatusC, []string{evVehicleConnect, evChargeStart}},

		{api.StatusB, api.StatusA, []string{evVehicleDisconnect}},
		{api.StatusB, api.StatusC, []string{evChargeStart}},

		{api.StatusC, api.StatusA, []string{evChargeStop, evVehicleDisconnect}},
		{api.StatusC, api.StatusB, []string{evChargeStop}},
	}

	for _, tc := range tc {
		ev := statusEvents(tc.from, tc.to)
		assert.Equalf(t, tc.events, ev, "from %s to %s got: %v", tc.from, tc.to, ev)
	}
}

// TestPhaseSwitchInterruption verifies that a charge interruption right after a phase
// switch is ignored while a real interruption is still detected
func TestPhaseSwitchInterruption(t *testing.T) {
	tc := []struct {
		desc       string
		since      time.Duration
		connection time.Duration
		expected   []api.ChargeStatus
	}{
		// the connection duration drops on interruption, too, and must not surface as intermediate disconnect
		{"during phase switch", phaseSwitchDuration - time.Second, time.Minute, nil},
		{"after phase switch", phaseSwitchDuration + time.Second, 2 * time.Hour, []api.ChargeStatus{api.StatusB}},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)

		charger := api.NewMockCharger(ctrl)
		charger.EXPECT().Status().Return(api.StatusB, nil)

		timer := api.NewMockConnectionTimer(ctrl)
		timer.EXPECT().ConnectionDuration().Return(tc.connection, nil)

		lp := &Loadpoint{
			log: util.NewLogger("foo"),
			charger: struct {
				*api.MockCharger
				*api.MockConnectionTimer
			}{charger, timer},
			status:            api.StatusC,
			phasesSwitched:    time.Now().Add(-tc.since),
			connectedDuration: time.Hour,
		}

		res, err := lp.getStatusChanges()
		require.NoError(t, err, tc.desc)
		assert.Equal(t, tc.expected, res, tc.desc)
		assert.Equal(t, api.StatusC, lp.GetStatus(), tc.desc)

		// connection duration is tracked even while the interruption is ignored
		assert.Equal(t, tc.connection, lp.connectedDuration, tc.desc)
	}
}
