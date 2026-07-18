package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestSetLimitLoadManagementThrottle verifies a plan-active charge whose current
// is capped by the circuit still applies the reduced current (throttle branch).
func TestSetLimitLoadManagementThrottle(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	Voltage = 230

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.minCurrent = 6
	lp.maxCurrent = 16
	lp.phases = 1
	lp.offeredCurrent = 0
	lp.wakeUpTimer = NewTimer()
	lp.planActive = true // deadline-bound

	charger := api.NewMockCharger(ctrl)
	lp.charger = charger
	charger.EXPECT().Enabled().Return(true, nil).AnyTimes()
	charger.EXPECT().Enable(gomock.Any()).Return(nil).AnyTimes()

	circuit := api.NewMockCircuit(ctrl)
	lp.circuit = circuit
	// circuit grants only 8A of the requested 16A; power is not the binding limit
	circuit.EXPECT().ValidateCurrent(gomock.Any(), 16.0).Return(8.0)
	circuit.EXPECT().ValidatePower(gomock.Any(), gomock.Any()).Return(1e6)

	// the reduced current is applied to the charger
	charger.EXPECT().MaxCurrent(int64(8)).Return(nil)

	require.NoError(t, lp.setLimit(16))
	require.Equal(t, 8.0, lp.offeredCurrent)
}
