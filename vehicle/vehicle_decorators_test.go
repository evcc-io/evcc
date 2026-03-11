package vehicle

import (
	"testing"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func baseVehicle() *Vehicle {
	return &Vehicle{
		embed: &embed{Title_: "test"},
		socG:  func() (float64, error) { return 42.0, nil },
	}
}

func TestDecorateVehicle_NoCapabilities(t *testing.T) {
	v := decorateVehicle(baseVehicle(), nil, nil, nil, nil, nil, nil, nil, nil, nil, nil)

	// base vehicle interface works
	soc, err := v.Soc()
	assert.NoError(t, err)
	assert.Equal(t, 42.0, soc)

	// no optional capabilities
	_, ok := api.Cap[api.ChargeState](v)
	assert.False(t, ok)
	_, ok = api.Cap[api.VehicleRange](v)
	assert.False(t, ok)
	_, ok = api.Cap[api.Resurrector](v)
	assert.False(t, ok)
}

func TestDecorateVehicle_AllCapabilities(t *testing.T) {
	v := decorateVehicle(baseVehicle(),
		func() (int64, error) { return 80, nil },                                              // socLimiter
		func() (api.ChargeStatus, error) { return api.StatusC, nil },                          // chargeState
		func() (int64, error) { return 300, nil },                                             // vehicleRange
		func() (float64, error) { return 12345.0, nil },                                       // vehicleOdometer
		func() (bool, error) { return true, nil },                                             // vehicleClimater
		func(int64) error { return nil },                                                      // currentController
		func() (float64, error) { return 16.0, nil },                                          // currentGetter
		func() (time.Time, error) { return time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), nil }, // vehicleFinishTimer
		func() error { return nil },                                                           // resurrector
		func(bool) error { return nil },                                                       // chargeController
	)

	// base interface
	soc, err := v.Soc()
	require.NoError(t, err)
	assert.Equal(t, 42.0, soc)

	// SocLimiter
	sl, ok := api.Cap[api.SocLimiter](v)
	require.True(t, ok, "SocLimiter")
	limit, err := sl.GetLimitSoc()
	assert.NoError(t, err)
	assert.Equal(t, int64(80), limit)

	// ChargeState
	cs, ok := api.Cap[api.ChargeState](v)
	require.True(t, ok, "ChargeState")
	status, err := cs.Status()
	assert.NoError(t, err)
	assert.Equal(t, api.StatusC, status)

	// VehicleRange
	vr, ok := api.Cap[api.VehicleRange](v)
	require.True(t, ok, "VehicleRange")
	rng, err := vr.Range()
	assert.NoError(t, err)
	assert.Equal(t, int64(300), rng)

	// VehicleOdometer
	vo, ok := api.Cap[api.VehicleOdometer](v)
	require.True(t, ok, "VehicleOdometer")
	odo, err := vo.Odometer()
	assert.NoError(t, err)
	assert.Equal(t, 12345.0, odo)

	// VehicleClimater
	vc, ok := api.Cap[api.VehicleClimater](v)
	require.True(t, ok, "VehicleClimater")
	active, err := vc.Climater()
	assert.NoError(t, err)
	assert.True(t, active)

	// CurrentController
	cc, ok := api.Cap[api.CurrentController](v)
	require.True(t, ok, "CurrentController")
	assert.NoError(t, cc.MaxCurrent(16))

	// CurrentGetter
	cg, ok := api.Cap[api.CurrentGetter](v)
	require.True(t, ok, "CurrentGetter")
	cur, err := cg.GetMaxCurrent()
	assert.NoError(t, err)
	assert.Equal(t, 16.0, cur)

	// VehicleFinishTimer
	ft, ok := api.Cap[api.VehicleFinishTimer](v)
	require.True(t, ok, "VehicleFinishTimer")
	fin, err := ft.FinishTime()
	assert.NoError(t, err)
	assert.Equal(t, 2026, fin.Year())

	// Resurrector
	r, ok := api.Cap[api.Resurrector](v)
	require.True(t, ok, "Resurrector")
	assert.NoError(t, r.WakeUp())

	// ChargeController
	chc, ok := api.Cap[api.ChargeController](v)
	require.True(t, ok, "ChargeController")
	assert.NoError(t, chc.ChargeEnable(true))
}

func TestDecorateVehicle_DependencyRules(t *testing.T) {
	t.Run("ChargeController requires ChargeState", func(t *testing.T) {
		v := decorateVehicle(baseVehicle(),
			nil,                             // socLimiter
			nil,                             // chargeState (nil!)
			nil,                             // vehicleRange
			nil,                             // vehicleOdometer
			nil,                             // vehicleClimater
			nil,                             // currentController
			nil,                             // currentGetter
			nil,                             // vehicleFinishTimer
			nil,                             // resurrector
			func(bool) error { return nil }, // chargeController (provided but should be pruned)
		)

		_, ok := api.Cap[api.ChargeController](v)
		assert.False(t, ok, "ChargeController should be pruned when ChargeState is nil")
	})

	t.Run("CurrentController requires ChargeState", func(t *testing.T) {
		v := decorateVehicle(baseVehicle(),
			nil,                              // socLimiter
			nil,                              // chargeState (nil!)
			nil,                              // vehicleRange
			nil,                              // vehicleOdometer
			nil,                              // vehicleClimater
			func(int64) error { return nil }, // currentController (provided but should be pruned)
			nil,                              // currentGetter
			nil,                              // vehicleFinishTimer
			nil,                              // resurrector
			nil,                              // chargeController
		)

		_, ok := api.Cap[api.CurrentController](v)
		assert.False(t, ok, "CurrentController should be pruned when ChargeState is nil")
	})

	t.Run("CurrentGetter requires CurrentController", func(t *testing.T) {
		v := decorateVehicle(baseVehicle(),
			nil, // socLimiter
			func() (api.ChargeStatus, error) { return api.StatusC, nil }, // chargeState
			nil, // vehicleRange
			nil, // vehicleOdometer
			nil, // vehicleClimater
			nil, // currentController (nil!)
			func() (float64, error) { return 16.0, nil }, // currentGetter (provided but should be pruned)
			nil, // vehicleFinishTimer
			nil, // resurrector
			nil, // chargeController
		)

		_, ok := api.Cap[api.CurrentGetter](v)
		assert.False(t, ok, "CurrentGetter should be pruned when CurrentController is nil")

		// ChargeState should still work though
		_, ok = api.Cap[api.ChargeState](v)
		assert.True(t, ok, "ChargeState should be present")
	})
}

func TestDecorateVehicle_PartialCapabilities(t *testing.T) {
	v := decorateVehicle(baseVehicle(),
		func() (int64, error) { return 80, nil }, // socLimiter
		nil,                                      // chargeState
		func() (int64, error) { return 300, nil }, // vehicleRange
		nil,                         // vehicleOdometer
		nil,                         // vehicleClimater
		nil,                         // currentController
		nil,                         // currentGetter
		nil,                         // vehicleFinishTimer
		func() error { return nil }, // resurrector
		nil,                         // chargeController
	)

	_, ok := api.Cap[api.SocLimiter](v)
	assert.True(t, ok, "SocLimiter should be present")

	_, ok = api.Cap[api.VehicleRange](v)
	assert.True(t, ok, "VehicleRange should be present")

	_, ok = api.Cap[api.Resurrector](v)
	assert.True(t, ok, "Resurrector should be present")

	_, ok = api.Cap[api.ChargeState](v)
	assert.False(t, ok, "ChargeState should not be present")

	_, ok = api.Cap[api.VehicleOdometer](v)
	assert.False(t, ok, "VehicleOdometer should not be present")
}
