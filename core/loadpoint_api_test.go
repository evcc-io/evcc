package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// vehicle limits must gate min/max current writes, otherwise setLimit dead-ends
// on every cycle and the charger is never updated (#32843)
func TestSetMinMaxCurrentVehicleLimits(t *testing.T) {
	ctrl := gomock.NewController(t)

	lp := NewLoadpoint(util.NewLogger("foo"), settings.NewDatabaseSettingsAdapter("foo"))
	lp.charger = api.NewMockCharger(ctrl)

	v := api.NewMockVehicle(ctrl)
	v.EXPECT().OnIdentified().Return(api.ActionConfig{MinCurrent: 8, MaxCurrent: 20}).AnyTimes()
	lp.vehicle = v

	// loadpoint max is 16A, vehicle caps at 20A- effective max is 16A
	require.Error(t, lp.SetMinCurrent(17))
	assert.Equal(t, 6.0, lp.GetMinCurrent())

	require.NoError(t, lp.SetMaxCurrent(24))

	// vehicle max 20A now wins over loadpoint max 24A
	require.Error(t, lp.SetMinCurrent(21))
	require.NoError(t, lp.SetMinCurrent(20))

	// vehicle min 8A raises the effective min above the loadpoint setting
	require.NoError(t, lp.SetMinCurrent(6))
	require.Error(t, lp.SetMaxCurrent(7))
	require.NoError(t, lp.SetMaxCurrent(8))
}
