package core

import (
	"errors"
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/coordinator"
	"github.com/evcc-io/evcc/core/soc"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPublishSocAndRange(t *testing.T) {
	ctrl := gomock.NewController(t)
	clck := clock.NewMock()

	charger := api.NewMockCharger(ctrl)
	charger.EXPECT().MaxCurrent(int64(maxA)).AnyTimes()
	charger.EXPECT().Enabled().Return(true, nil).AnyTimes()

	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Title().Return("target").AnyTimes()
	vehicle.EXPECT().Capacity().AnyTimes()
	vehicle.EXPECT().Phases().AnyTimes()
	vehicle.EXPECT().OnIdentified().AnyTimes()

	log := util.NewLogger("foo")
	lp := &Loadpoint{
		log:           log,
		bus:           evbus.New(),
		clock:         clck,
		charger:       charger,
		vehicle:       vehicle,
		chargeMeter:   &Null{}, // silence nil panics
		chargeRater:   &Null{}, // silence nil panics
		chargeTimer:   &Null{}, // silence nil panics
		socEstimator:  soc.NewEstimator(log, charger, vehicle, false),
		sessionEnergy: NewEnergyMetrics(),
		MinCurrent:    minA,
		MaxCurrent:    maxA,
		phases:        1,
		mode:          api.ModeNow,
	}

	// populate channels
	x, y, z := createChannels(t)
	attachChannels(lp, x, y, z)

	assert.Empty(t, lp.socUpdated)

	tc := []struct {
		status  api.ChargeStatus
		allowed bool
	}{
		{api.StatusB, false},
		{api.StatusC, true},
	}

	for _, tc := range tc {
		clck.Add(time.Hour)
		lp.status = tc.status

		assert.True(t, lp.vehicleSocPollAllowed())
		vehicle.EXPECT().Soc().Return(0.0, errors.New("foo"))
		lp.publishSocAndRange()

		clck.Add(time.Second)
		assert.Equal(t, tc.allowed, lp.vehicleSocPollAllowed())
		if tc.allowed {
			vehicle.EXPECT().Soc().Return(0.0, errors.New("foo"))
		}
		lp.publishSocAndRange()
	}
}

func TestVehicleDetectByID(t *testing.T) {
	ctrl := gomock.NewController(t)

	v1 := api.NewMockVehicle(ctrl)
	v2 := api.NewMockVehicle(ctrl)

	type testcase struct {
		string
		id, i1, i2 string
		res        api.Vehicle
		prepare    func(testcase)
	}
	tc := []testcase{
		{"1/_/_->0", "1", "", "", nil, func(tc testcase) {
			v1.EXPECT().Identifiers().Return(nil)
			v2.EXPECT().Identifiers().Return(nil)
			v1.EXPECT().Identifiers().Return(nil)
			v2.EXPECT().Identifiers().Return(nil)
		}},
		{"1/1/2->1", "1", "1", "2", v1, func(tc testcase) {
			v1.EXPECT().Identifiers().Return([]string{tc.i1})
		}},
		{"2/1/2->2", "2", "1", "2", v2, func(tc testcase) {
			v1.EXPECT().Identifiers().Return([]string{tc.i1})
			v2.EXPECT().Identifiers().Return([]string{tc.i2})
		}},
		{"11/1*/2->1", "11", "1*", "2", v1, func(tc testcase) {
			v1.EXPECT().Identifiers().Return([]string{tc.i1})
			v2.EXPECT().Identifiers().Return([]string{tc.i2})
			v1.EXPECT().Identifiers().Return([]string{tc.i1})
			// v2.EXPECT().Identifiers().Return([]string{tc.i2})
		}},
		{"22/1*/2*->2", "22", "1*", "2*", v2, func(tc testcase) {
			v1.EXPECT().Identifiers().Return([]string{tc.i1})
			v2.EXPECT().Identifiers().Return([]string{tc.i2})
			v1.EXPECT().Identifiers().Return([]string{tc.i1})
			v2.EXPECT().Identifiers().Return([]string{tc.i2})
		}},
		{"2/_/*->2", "2", "", "*", v2, func(tc testcase) {
			v1.EXPECT().Identifiers().Return(nil)
			v2.EXPECT().Identifiers().Return([]string{tc.i2})
			v1.EXPECT().Identifiers().Return(nil)
			v2.EXPECT().Identifiers().Return([]string{tc.i2})
		}},
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)

		lp := &Loadpoint{
			log: util.NewLogger("foo"),
		}

		lp.coordinator = coordinator.NewAdapter(lp, coordinator.New(util.NewLogger("foo"), []api.Vehicle{v1, v2}))

		if tc.prepare != nil {
			tc.prepare(tc)
		}

		if res := lp.selectVehicleByID(tc.id); tc.res != res {
			t.Errorf("expected %v, got %v", tc.res, res)
		}
	}
}

func TestDefaultVehicle(t *testing.T) {
	ctrl := gomock.NewController(t)

	mode := api.ModePV
	current := 66.6

	dflt := api.NewMockVehicle(ctrl)
	dflt.EXPECT().Title().Return("default").AnyTimes()
	dflt.EXPECT().Icon().Return("").AnyTimes()
	dflt.EXPECT().Capacity().AnyTimes()
	dflt.EXPECT().Phases().AnyTimes()
	dflt.EXPECT().OnIdentified().Return(api.ActionConfig{
		Mode:       mode,
		MinCurrent: current,
	}).AnyTimes()

	vehicle := api.NewMockVehicle(ctrl)
	vehicle.EXPECT().Title().Return("target").AnyTimes()
	vehicle.EXPECT().Icon().Return("").AnyTimes()
	vehicle.EXPECT().Capacity().AnyTimes()
	vehicle.EXPECT().Phases().AnyTimes()
	vehicle.EXPECT().OnIdentified().AnyTimes()

	lp := NewLoadpoint(util.NewLogger("foo"), nil)
	lp.Mode_ = api.ModeOff // ondisconnect
	lp.defaultVehicle = dflt

	// populate channels
	x, y, z := createChannels(t)
	attachChannels(lp, x, y, z)

	title := func(v api.Vehicle) string {
		if v == nil {
			return "<nil>"
		}
		return v.Title()
	}

	// non-default vehicle identified
	lp.setActiveVehicle(vehicle)
	assert.Equal(t, vehicle, lp.vehicle, "expected vehicle "+title(vehicle))
	assert.Equal(t, 6.0, lp.effectiveMinCurrent(), "current")

	// non-default vehicle disconnected
	lp.evVehicleDisconnectHandler()
	assert.Equal(t, dflt, lp.vehicle, "expected default vehicle")
	assert.Equal(t, mode, lp.GetMode(), "mode")
	assert.Equal(t, current, lp.effectiveMinCurrent(), "current")

	// default vehicle disconnected and reconnected
	lp.evVehicleDisconnectHandler()
	assert.Equal(t, mode, lp.GetMode(), "mode")
	assert.Equal(t, current, lp.effectiveMinCurrent(), "current")

	// set non-default vehicle during disconnect - should be default on connect
	lp.tasks.Clear()
	lp.evVehicleConnectHandler()
	assert.Equal(t, dflt, lp.vehicle, "expected default vehicle")
	assert.Equal(t, 1, lp.tasks.Size(), "task queue length")

	// guest connected
	lp.setActiveVehicle(nil)
	assert.Nil(t, lp.vehicle, "expected no vehicle")
}

func TestReconnectVehicle(t *testing.T) {
	tc := []struct {
		name      string
		vehicleId []string
	}{
		{"without vehicle id", nil},
		{"with vehicle id", []string{"foo"}},
	}

	for _, tc := range tc {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			clck := clock.NewMock()

			type vehicleT struct {
				*api.MockVehicle
				*api.MockChargeState
			}

			vehicle := &vehicleT{api.NewMockVehicle(ctrl), api.NewMockChargeState(ctrl)}
			vehicle.MockVehicle.EXPECT().Title().Return("vehicle").AnyTimes()
			vehicle.MockVehicle.EXPECT().Icon().Return("").AnyTimes()
			vehicle.MockVehicle.EXPECT().Capacity().AnyTimes()
			vehicle.MockVehicle.EXPECT().Phases().AnyTimes()
			vehicle.MockVehicle.EXPECT().OnIdentified().AnyTimes()
			vehicle.MockVehicle.EXPECT().Identifiers().AnyTimes().Return(tc.vehicleId)
			vehicle.MockVehicle.EXPECT().Soc().Return(0.0, nil).AnyTimes()

			charger := api.NewMockCharger(ctrl)
			charger.EXPECT().Status().Return(api.StatusB, nil).AnyTimes()

			lp := &Loadpoint{
				log:           util.NewLogger("foo"),
				bus:           evbus.New(),
				clock:         clck,
				charger:       charger,
				chargeMeter:   &Null{}, // silence nil panics
				chargeRater:   &Null{}, // silence nil panics
				chargeTimer:   &Null{}, // silence nil panics
				wakeUpTimer:   NewTimer(),
				sessionEnergy: NewEnergyMetrics(),
				MinCurrent:    minA,
				MaxCurrent:    maxA,
				phases:        1,
				mode:          api.ModeNow,
			}

			lp.coordinator = coordinator.NewAdapter(lp, coordinator.New(util.NewLogger("foo"), []api.Vehicle{vehicle}))

			attachListeners(t, lp)

			// mode now
			charger.EXPECT().MaxCurrent(int64(maxA))
			// sync charger
			charger.EXPECT().Enabled().Return(true, nil)

			// vehicle not updated yet
			vehicle.MockChargeState.EXPECT().Status().Return(api.StatusA, nil)

			lp.Update(0, false, false, false, 0, nil, nil)
			ctrl.Finish()

			// detection started
			assert.Equal(t, lp.clock.Now(), lp.vehicleDetect, "vehicle detection not started")

			// vehicle not detected yet
			assert.Nil(t, lp.vehicle, "vehicle should be <nil>")

			// sync charger
			charger.EXPECT().Enabled().Return(true, nil)
			// vehicle not updated yet
			vehicle.MockChargeState.EXPECT().Status().Return(api.StatusB, nil)

			lp.Update(0, false, false, false, 0, nil, nil)
			ctrl.Finish()

			// vehicle detected
			assert.Equal(t, vehicle, lp.vehicle, "vehicle should be detected")
		})
	}
}
