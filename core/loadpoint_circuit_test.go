package core

import (
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/core/wrapper"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// meterlessLoadpoint returns a loadpoint with a wrapped charge meter, i.e. without
// measured currents
func meterlessLoadpoint(chg api.Charger, c api.Circuit, status api.ChargeStatus, offered float64) *Loadpoint {
	return &Loadpoint{
		log:            util.NewLogger("lp"),
		bus:            evbus.New(),
		clock:          clock.New(),
		charger:        chg,
		chargeMeter:    new(wrapper.ChargeMeter),
		circuit:        c,
		wakeUpTimer:    NewTimer(),
		status:         status,
		offeredCurrent: offered,
		enabled:        true,
		minCurrent:     6,
		maxCurrent:     16,
		phases:         1,
	}
}

// siteCycle samples the loadpoints and updates the circuit, in the order site.update does
func siteCycle(t *testing.T, c api.Circuit, lps ...*Loadpoint) {
	t.Helper()

	loads := make([]api.CircuitLoad, 0, len(lps))
	for _, lp := range lps {
		lp.UpdateChargePowerAndCurrents()
		loads = append(loads, lp)
	}

	require.NoError(t, c.Update(loads))
}

// a loadpoint that has been offered current but is not charging yet must not be
// blocked by its own offer
func TestSetLimitDeadlockPrevention(t *testing.T) {
	Voltage = 230
	ctrl := gomock.NewController(t)

	c, err := circuit.New(util.NewLogger("circuit"), "main", 16, 0, nil, 0)
	require.NoError(t, err)

	chg := api.NewMockCharger(ctrl)
	lp := meterlessLoadpoint(chg, c, api.StatusB, 10)

	siteCycle(t, c, lp)
	assert.Equal(t, 0.0, c.GetMaxPhaseCurrent(), "not charging, hence no load")

	chg.EXPECT().MaxCurrent(int64(16)).Return(nil)

	require.NoError(t, lp.setLimit(16))
	assert.Equal(t, 16.0, lp.offeredCurrent)
}

// the circuit limit is still enforced when the charger starts charging after the
// circuit has been updated
func TestSetLimitWithMeterlessCircuitAndMeterlessCharger(t *testing.T) {
	Voltage = 230
	ctrl := gomock.NewController(t)

	c, err := circuit.New(util.NewLogger("circuit"), "main", 12, 0, nil, 0)
	require.NoError(t, err)

	chg := api.NewMockCharger(ctrl)
	lp := meterlessLoadpoint(chg, c, api.StatusB, 10)

	siteCycle(t, c, lp)

	chg.EXPECT().Status().Return(api.StatusC, nil)
	_, err = lp.updateChargerStatus()
	require.NoError(t, err)

	chg.EXPECT().MaxCurrent(int64(12)).Return(nil)

	require.NoError(t, lp.setLimit(16))
	assert.Equal(t, 12.0, lp.offeredCurrent)
}

// a loadpoint that has been offered current but never draws it must not block
// another loadpoint on the same circuit
func TestSetLimitMeterlessDoesNotBlockOtherLoadpoint(t *testing.T) {
	Voltage = 230
	ctrl := gomock.NewController(t)

	c, err := circuit.New(util.NewLogger("circuit"), "main", 16, 0, nil, 0)
	require.NoError(t, err)

	lp1 := meterlessLoadpoint(api.NewMockCharger(ctrl), c, api.StatusB, 16)
	chg2 := api.NewMockCharger(ctrl)
	lp2 := meterlessLoadpoint(chg2, c, api.StatusB, 0)

	siteCycle(t, c, lp1, lp2)

	chg2.EXPECT().MaxCurrent(int64(16)).Return(nil)

	require.NoError(t, lp2.setLimit(16))
	assert.Equal(t, 16.0, lp2.offeredCurrent)
}

// once both loadpoints charge the circuit limit applies again
func TestSetLimitMeterlessBothCharging(t *testing.T) {
	Voltage = 230
	ctrl := gomock.NewController(t)

	c, err := circuit.New(util.NewLogger("circuit"), "main", 16, 0, nil, 0)
	require.NoError(t, err)

	lp1 := meterlessLoadpoint(api.NewMockCharger(ctrl), c, api.StatusC, 16)
	chg2 := api.NewMockCharger(ctrl)
	lp2 := meterlessLoadpoint(chg2, c, api.StatusC, 16)

	siteCycle(t, c, lp1, lp2)
	assert.Equal(t, 32.0, c.GetMaxPhaseCurrent())

	chg2.EXPECT().Enable(false).Return(nil)

	require.NoError(t, lp2.setLimit(16))
	assert.Equal(t, 0.0, lp2.offeredCurrent)
}
