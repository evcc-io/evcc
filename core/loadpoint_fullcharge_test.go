package core

import (
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/session"
	"github.com/evcc-io/evcc/core/vehicle"
	serverdb "github.com/evcc-io/evcc/db"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func fullChargeLoadpoint(t *testing.T, fc api.FullCharge, last time.Time) (*Loadpoint, *clock.Mock, vehicle.API) {
	t.Helper()

	config.Reset()
	t.Cleanup(config.Reset)

	ctrl := gomock.NewController(t)
	v := api.NewMockVehicle(ctrl)
	v.EXPECT().GetTitle().Return("lfp").AnyTimes()

	const name = "lfp"
	require.NoError(t, config.Vehicles().Add(
		config.NewStaticDevice(config.Named{Name: name}, api.Vehicle(v)),
	))

	settings.SetJson("vehicle."+name+"."+keys.FullCharge, fc)
	settings.SetTime("vehicle."+name+"."+keys.FullChargeTime, last)
	settings.SetInt("vehicle."+name+"."+keys.LimitSoc, 80)

	clk := clock.NewMock()
	clk.Set(time.Date(2026, 11, 3, 12, 0, 0, 0, time.UTC))

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.clock = clk
	lp.vehicle = v

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

func TestFullChargeNotRecordedWhenDisabledByEvcc(t *testing.T) {
	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	lp, clk, vs := fullChargeLoadpoint(t, api.FullCharge{Days: 14}, start)

	// no surplus: evcc disabled the charger, the vehicle did not stop on its own
	lp.vehicleSoc = 100
	lp.enabled = false
	lp.status = api.StatusB
	lp.updateFullCharge()
	clk.Add(2 * fullChargeHold)
	lp.updateFullCharge()

	assert.True(t, lp.fullChargeActive)
	assert.Equal(t, start, vs.GetFullChargeTime())
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

func TestFullChargeEnergy(t *testing.T) {
	start := time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)
	lp, clk, _ := fullChargeLoadpoint(t, api.FullCharge{Energy: 190}, start)

	var err error
	serverdb.Instance, err = serverdb.New("sqlite", ":memory:")
	require.NoError(t, err)
	lp.db, err = session.NewStore("foo", serverdb.Instance)
	require.NoError(t, err)

	for _, s := range []session.Session{
		{Vehicle: "lfp", Created: start.Add(-time.Hour), ChargedEnergy: 500}, // before the last full charge
		{Vehicle: "lfp", Created: start.Add(time.Hour), ChargedEnergy: 100},
		{Vehicle: "other", Created: start.Add(time.Hour), ChargedEnergy: 500}, // other vehicle
		{Vehicle: "lfp", Created: start.Add(2 * time.Hour), ChargedEnergy: 80},
	} {
		lp.db.Persist(&s)
	}

	lp.updateFullCharge()
	assert.False(t, lp.fullChargeActive, "180kWh")

	lp.db.Persist(&session.Session{Vehicle: "lfp", Created: start.Add(3 * time.Hour), ChargedEnergy: 15})

	// cached until the interval elapsed
	lp.updateFullCharge()
	assert.False(t, lp.fullChargeActive)

	clk.Add(fullChargeEnergyInterval)
	lp.updateFullCharge()
	assert.True(t, lp.fullChargeActive, "195kWh")
}
