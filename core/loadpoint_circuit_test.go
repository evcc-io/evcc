package core

import (
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// A charging loadpoint measuring below its offer on a full circuit keeps its min current
func TestSetLimitCircuitKeepsMinCurrent(t *testing.T) {
	for _, tc := range []struct {
		name     string
		headroom float64 // circuit headroom beyond the measured current
		status   api.ChargeStatus
		disable  bool
	}{
		{"within limits", 0, api.StatusC, false},
		{"over limit", -2, api.StatusC, true},
		{"not charging", 0, api.StatusB, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			charger := api.NewMockCharger(ctrl)
			circuit := api.NewMockCircuit(ctrl)

			circuit.EXPECT().ValidateCurrent(gomock.Any(), gomock.Any()).DoAndReturn(func(old, new float64) float64 {
				return min(new, max(0, old+tc.headroom))
			}).AnyTimes()
			circuit.EXPECT().ValidatePower(gomock.Any(), gomock.Any()).DoAndReturn(func(_, new float64) float64 {
				return new
			}).AnyTimes()

			lp := &Loadpoint{
				log:            util.NewLogger("foo"),
				bus:            evbus.New(),
				clock:          clock.New(),
				charger:        charger,
				circuit:        circuit,
				wakeUpTimer:    NewTimer(),
				minCurrent:     6,
				maxCurrent:     16,
				phases:         3,
				status:         tc.status,
				enabled:        true,
				offeredCurrent: 6,
				chargeCurrents: []float64{5.9, 5.9, 5.9},
			}

			if tc.disable {
				charger.EXPECT().Enable(false).Return(nil)
			}

			require.NoError(t, lp.setLimit(16))
			require.Equal(t, !tc.disable, lp.enabled)
		})
	}
}
