package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestMinSocNotReachedEnergyFallback covers the energy fallback path of
// minSocNotReached() that is used when no vehicle soc is available
// (vehicleSoc == 0). Capacity() is in kWh while getChargedEnergy() is in Wh, so
// the required minimum energy must be converted to Wh before the comparison.
//
// 10 kWh vehicle, minSoc 50%, charge efficiency 0.85:
//
//	minEnergy = 10 * 1e3 * 50/100 / 0.85 = 5882.35 Wh
//
// With only 3000 Wh charged so far the minimum is not reached, so
// minSocNotReached() must return true.
func TestMinSocNotReachedEnergyFallback(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	ctrl := gomock.NewController(t)

	const vehicleName = "minsocvehicle"
	const capacityKWh = 10.0
	const minSocPct = 50

	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Capacity().Return(capacityKWh).AnyTimes()
	vehicle.EXPECT().GetTitle().Return("minsoc").AnyTimes()

	// register the vehicle so its persisted settings (min soc) are used
	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: vehicleName}, api.Vehicle(vehicle)),
	))
	settings.SetInt("vehicle."+vehicleName+"."+keys.MinSoc, int64(minSocPct))

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.vehicle = vehicle
	lp.vehicleSoc = 0 // no soc reading- exercise the energy fallback

	lp.energyMetrics.totalKWh = 3.0 // 3000 Wh charged so far
	require.Equal(t, 3000.0, lp.getChargedEnergy())

	wantMinEnergyWh := capacityKWh * 1e3 * float64(minSocPct) / 100 / soc.ChargeEfficiency
	require.InDelta(t, 5882.35, wantMinEnergyWh, 0.1)

	// 3000 Wh < 5882 Wh required => minimum not reached
	require.True(t, lp.minSocNotReached())
}
