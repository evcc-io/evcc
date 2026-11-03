package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func fullChargeVehicle(t *testing.T, name string, fc api.FullCharge, last time.Time) *api.MockVehicle {
	t.Helper()

	v := api.NewMockVehicle(gomock.NewController(t))
	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: name}, api.Vehicle(v)),
	))

	settings.SetJson("vehicle."+name+"."+keys.FullCharge, fc)
	settings.SetTime("vehicle."+name+"."+keys.FullChargeTime, last)
	settings.SetFloat("vehicle."+name+"."+keys.FullChargeEnergy, 0)
	settings.SetInt("vehicle."+name+"."+keys.LimitSoc, 80)

	return v
}

func fullChargeLoadpoint(t *testing.T, fc api.FullCharge, last time.Time) (*Loadpoint, *clock.Mock, vehicle.API) {
	t.Helper()

	config.Reset()
	t.Cleanup(config.Reset)

	v := fullChargeVehicle(t, "lfp", fc, last)

	clk := clock.NewMock()
	clk.Set(time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC))

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.clock = clk
	lp.vehicle = v
	lp.status = api.StatusB

	return lp, clk, vehicle.Settings(lp.log, v)
}

func TestFullChargeDisabled(t *testing.T) {
	lp, clk, _ := fullChargeLoadpoint(t, api.FullCharge{}, time.Time{})
	clk.Add(1000 * 24 * time.Hour)

	lp.updateFullCharge()
	assert.False(t, lp.fullChargeActive)
	assert.Equal(t, 80, lp.effectiveLimitSoc())
}

func TestFullChargeDays(t *testing.T) {
	lp, clk, vs := fullChargeLoadpoint(t, api.FullCharge{Days: 14}, time.Time{})

	// first evaluation sets the reference
	lp.updateFullCharge()
	assert.False(t, lp.fullChargeActive)
	assert.Equal(t, clk.Now(), vs.GetFullChargeTime())

	clk.Add(13 * 24 * time.Hour)
	lp.updateFullCharge()
	assert.False(t, lp.fullChargeActive)

	clk.Add(24 * time.Hour)
	lp.updateFullCharge()
	assert.True(t, lp.fullChargeActive)
	assert.Equal(t, 100, lp.effectiveLimitSoc(), "full charge overrides vehicle limit")

	lp.limitSoc = 90
	assert.Equal(t, 100, lp.effectiveLimitSoc(), "full charge overrides session limit")
}

func TestFullChargeCompletion(t *testing.T) {
	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	lp, clk, vs := fullChargeLoadpoint(t, api.FullCharge{Days: 14}, start)

	lp.updateFullCharge()
	require.True(t, lp.fullChargeActive)

	// still charging at 100%: balancing
	lp.vehicleSoc = 100
	lp.enabled = true
	lp.status = api.StatusC
	lp.updateFullCharge()
	assert.True(t, lp.fullChargeActive)

	// stopped on its own, hold time running
	lp.status = api.StatusB
	lp.updateFullCharge()
	clk.Add(fullChargeHold - time.Minute)
	lp.updateFullCharge()
	assert.True(t, lp.fullChargeActive)
	assert.Equal(t, start, vs.GetFullChargeTime())

	// resumes charging, hold time restarts
	lp.status = api.StatusC
	lp.updateFullCharge()
	lp.status = api.StatusB
	lp.updateFullCharge()
	clk.Add(fullChargeHold - time.Minute)
	lp.updateFullCharge()
	assert.True(t, lp.fullChargeActive)

	clk.Add(time.Minute)
	lp.updateFullCharge()
	assert.False(t, lp.fullChargeActive)
	assert.Equal(t, clk.Now(), vs.GetFullChargeTime())
	assert.Equal(t, 80, lp.effectiveLimitSoc())

	// remaining at 100% does not move the date
	recorded := vs.GetFullChargeTime()
	clk.Add(2 * fullChargeHold)
	lp.updateFullCharge()
	assert.Equal(t, recorded, vs.GetFullChargeTime())
}

func TestFullChargeIdleAtFull(t *testing.T) {
	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	lp, clk, vs := fullChargeLoadpoint(t, api.FullCharge{Days: 14}, start)

	// connected at 100% without charging, e.g. after a public charger
	lp.vehicleSoc = 100
	lp.enabled = true
	lp.updateFullCharge()
	clk.Add(2 * fullChargeHold)
	lp.updateFullCharge()

	assert.True(t, lp.fullChargeActive)
	assert.Equal(t, start, vs.GetFullChargeTime())
}

func TestFullChargeNotRecordedWhenDisabledByEvcc(t *testing.T) {
	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	lp, clk, vs := fullChargeLoadpoint(t, api.FullCharge{Days: 14}, start)

	lp.vehicleSoc = 100
	lp.status = api.StatusC
	lp.updateFullCharge()

	// no surplus: evcc disabled the charger, the vehicle did not stop on its own
	lp.enabled = false
	lp.status = api.StatusB
	lp.updateFullCharge()
	clk.Add(2 * fullChargeHold)
	lp.updateFullCharge()

	assert.True(t, lp.fullChargeActive)
	assert.Equal(t, start, vs.GetFullChargeTime())
}

func TestFullChargeVehicleChange(t *testing.T) {
	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	lp, clk, vs := fullChargeLoadpoint(t, api.FullCharge{Days: 14}, start)

	lp.vehicleSoc = 100
	lp.enabled = true
	lp.status = api.StatusC
	lp.updateFullCharge()
	lp.status = api.StatusB
	lp.updateFullCharge()

	// another vehicle takes over the loadpoint during the hold time
	other := fullChargeVehicle(t, "other", api.FullCharge{Days: 14}, start)
	lp.vehicle = other
	clk.Add(2 * fullChargeHold)
	lp.updateFullCharge()

	assert.Equal(t, start, vs.GetFullChargeTime())
	assert.Equal(t, start, vehicle.Settings(lp.log, other).GetFullChargeTime(), "no charge seen for this vehicle")
}

func TestFullChargeEnergy(t *testing.T) {
	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	lp, _, vs := fullChargeLoadpoint(t, api.FullCharge{Energy: 190}, start)

	// energy charged before the vehicle was identified is not counted
	lp.energyMetrics.Update(20)
	lp.updateFullCharge()
	assert.Zero(t, vs.GetFullChargeEnergy())

	lp.energyMetrics.Update(120)
	lp.updateFullCharge()
	assert.InDelta(t, 100, vs.GetFullChargeEnergy(), 1e-6)
	assert.False(t, lp.fullChargeActive)

	// a new session starts from zero
	lp.energyMetrics.Reset()
	lp.energyMetrics.Update(95)
	lp.updateFullCharge()
	assert.InDelta(t, 195, vs.GetFullChargeEnergy(), 1e-6)
	assert.True(t, lp.fullChargeActive)

	// a recorded full charge resets the energy
	vs.SetFullChargeTime(time.Now())
	assert.Zero(t, vs.GetFullChargeEnergy())
}

func TestFullChargeHeating(t *testing.T) {
	lp, clk, _ := fullChargeLoadpoint(t, api.FullCharge{Days: 1}, time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))

	ctrl := gomock.NewController(t)
	charger := api.NewMockCharger(ctrl)
	fd := api.NewMockFeatureDescriber(ctrl)
	fd.EXPECT().Features().Return([]api.Feature{api.Heating}).AnyTimes()
	lp.charger = struct {
		api.Charger
		api.FeatureDescriber
	}{charger, fd}

	clk.Add(24 * time.Hour)
	lp.updateFullCharge()
	assert.False(t, lp.fullChargeActive, "soc is a temperature for heating devices")
}
