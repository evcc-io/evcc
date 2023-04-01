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
	"github.com/evcc-io/evcc/mock"
	"github.com/evcc-io/evcc/util"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestPublishSocAndRange(t *testing.T) {
	ctrl := gomock.NewController(t)
	clck := clock.NewMock()

	charger := mock.NewMockCharger(ctrl)
	charger.EXPECT().MaxCurrent(int64(maxA)).AnyTimes()
	charger.EXPECT().Enabled().Return(true, nil).AnyTimes()

	vehicle := mock.NewMockVehicle(ctrl)
	vehicle.EXPECT().Title().Return("target").AnyTimes()
	vehicle.EXPECT().Capacity().AnyTimes()
	vehicle.EXPECT().Phases().AnyTimes()
	vehicle.EXPECT().OnIdentified().AnyTimes()

	log := util.NewLogger("foo")
	lp := &Loadpoint{
		log:            log,
		bus:            evbus.New(),
		clock:          clck,
		charger:        charger,
		defaultVehicle: vehicle,
		chargeMeter:    &Null{}, // silence nil panics
		chargeRater:    &Null{}, // silence nil panics
		chargeTimer:    &Null{}, // silence nil panics
		socEstimator:   soc.NewEstimator(log, charger, vehicle, false),
		MinCurrent:     minA,
		MaxCurrent:     maxA,
		phases:         1,
		Mode:           api.ModeNow,
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

	v1 := mock.NewMockVehicle(ctrl)
	v2 := mock.NewMockVehicle(ctrl)

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
	minsoc := 20
	targetsoc := 80

	dflt := mock.NewMockVehicle(ctrl)
	dflt.EXPECT().Title().Return("default").AnyTimes()
	dflt.EXPECT().Icon().Return("").AnyTimes()
	dflt.EXPECT().Capacity().AnyTimes()
	dflt.EXPECT().Phases().AnyTimes()
	dflt.EXPECT().OnIdentified().Return(api.ActionConfig{
		Mode:      &mode,
		MinSoc:    &minsoc,
		TargetSoc: &targetsoc,
	}).AnyTimes()

	vehicle := mock.NewMockVehicle(ctrl)
	vehicle.EXPECT().Title().Return("target").AnyTimes()
	vehicle.EXPECT().Icon().Return("").AnyTimes()
	vehicle.EXPECT().Capacity().AnyTimes()
	vehicle.EXPECT().Phases().AnyTimes()
	vehicle.EXPECT().OnIdentified().AnyTimes()

	lp := NewLoadpoint(util.NewLogger("foo"))
	lp.defaultVehicle = dflt

	// ondisconnect
	off := api.ModeOff
	zero := 0
	hundred := 100
	onDisconnect := api.ActionConfig{
		Mode:       &off,
		MinCurrent: &lp.MinCurrent,
		MaxCurrent: &lp.MaxCurrent,
		MinSoc:     &zero,
		TargetSoc:  &hundred,
		Priority:   &zero,
	}

	lp.collectDefaults()
	assert.Equal(t, lp.onDisconnect, onDisconnect)

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

	// non-default vehicle disconnected
	lp.evVehicleDisconnectHandler()
	assert.Equal(t, dflt, lp.vehicle, "expected default vehicle")

	// default vehicle disconnected
	lp.ResetOnDisconnect = true
	lp.evVehicleDisconnectHandler()
	assert.Equal(t, mode, lp.GetMode(), "mode")
	assert.Equal(t, minsoc, lp.GetMinSoc(), "minsoc")
	assert.Equal(t, targetsoc, lp.GetTargetSoc(), "targetsoc")
	assert.Equal(t, lp.onDisconnect, onDisconnect, "ondisconnect must remain untouched")

	// set non-default vehicle during disconnect - should be default on connect
	lp.tasks.Clear()
	lp.evVehicleConnectHandler()
	assert.Equal(t, dflt, lp.vehicle, "expected default vehicle")
	assert.Equal(t, 1, lp.tasks.Size(), "task queue length")

	// guest connected
	lp.setActiveVehicle(nil)
	assert.Nil(t, lp.vehicle, "expected no vehicle")
}

func TestApplyVehicleDefaults(t *testing.T) {
	ctrl := gomock.NewController(t)

	newConfig := func(mode api.ChargeMode, minCurrent, maxCurrent float64, minSoc, targetSoc int) api.ActionConfig {
		return api.ActionConfig{
			Mode:       &mode,
			MinCurrent: &minCurrent,
			MaxCurrent: &maxCurrent,
			MinSoc:     &minSoc,
			TargetSoc:  &targetSoc,
		}
	}

	assertConfig := func(lp *Loadpoint, conf api.ActionConfig) {
		assert.Equal(t, *conf.Mode, lp.Mode)
		assert.Equal(t, *conf.MinCurrent, lp.MinCurrent)
		assert.Equal(t, *conf.MaxCurrent, lp.MaxCurrent)
		assert.Equal(t, *conf.MinSoc, lp.Soc.min)
		assert.Equal(t, *conf.TargetSoc, lp.Soc.target)
	}

	// onIdentified config
	oi := newConfig(api.ModePV, 7, 15, 1, 99)

	// onDefault config
	od := newConfig(api.ModeOff, 6, 16, 2, 98)

	vehicle := mock.NewMockVehicle(ctrl)
	vehicle.EXPECT().Title().Return("it's me").AnyTimes()
	vehicle.EXPECT().Icon().Return("").AnyTimes()
	vehicle.EXPECT().Capacity().AnyTimes()
	vehicle.EXPECT().Phases().AnyTimes()
	vehicle.EXPECT().OnIdentified().Return(oi).AnyTimes()

	lp := NewLoadpoint(util.NewLogger("foo"))

	// populate channels
	x, y, z := createChannels(t)
	attachChannels(lp, x, y, z)

	lp.onDisconnect = od
	lp.ResetOnDisconnect = true

	// check loadpoint default currents can't be violated
	lp.applyAction(newConfig(*od.Mode, 5, 17, *od.MinSoc, *od.TargetSoc))
	assertConfig(lp, od)

	// vehicle identified
	lp.setActiveVehicle(vehicle)
	assertConfig(lp, oi)

	// vehicle disconnected
	vehicle.EXPECT().Phases().AnyTimes()
	lp.evVehicleDisconnectHandler()
	assertConfig(lp, od)

	// identify vehicle by id
	charger := struct {
		*mock.MockCharger
		*mock.MockIdentifier
	}{
		MockCharger:    mock.NewMockCharger(ctrl),
		MockIdentifier: mock.NewMockIdentifier(ctrl),
	}

	lp.charger = charger
	lp.coordinator = coordinator.NewAdapter(lp, coordinator.New(util.NewLogger("foo"), []api.Vehicle{vehicle}))

	const id = "don't call me stacey"
	charger.MockIdentifier.EXPECT().Identify().Return(id, nil)
	vehicle.EXPECT().Identifiers().Return([]string{id})

	lp.identifyVehicle()
	assertConfig(lp, oi)
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
				*mock.MockVehicle
				*mock.MockChargeState
			}

			vehicle := &vehicleT{mock.NewMockVehicle(ctrl), mock.NewMockChargeState(ctrl)}
			vehicle.MockVehicle.EXPECT().Title().Return("vehicle").AnyTimes()
			vehicle.MockVehicle.EXPECT().Icon().Return("").AnyTimes()
			vehicle.MockVehicle.EXPECT().Capacity().AnyTimes()
			vehicle.MockVehicle.EXPECT().Phases().AnyTimes()
			vehicle.MockVehicle.EXPECT().OnIdentified().AnyTimes()
			vehicle.MockVehicle.EXPECT().Identifiers().AnyTimes().Return(tc.vehicleId)
			vehicle.MockVehicle.EXPECT().Soc().Return(0.0, nil).AnyTimes()

			charger := mock.NewMockCharger(ctrl)
			charger.EXPECT().Status().Return(api.StatusB, nil).AnyTimes()

			lp := &Loadpoint{
				log:         util.NewLogger("foo"),
				bus:         evbus.New(),
				clock:       clck,
				charger:     charger,
				chargeMeter: &Null{}, // silence nil panics
				chargeRater: &Null{}, // silence nil panics
				chargeTimer: &Null{}, // silence nil panics
				wakeUpTimer: NewTimer(),
				MinCurrent:  minA,
				MaxCurrent:  maxA,
				phases:      1,
				Mode:        api.ModeNow,
			}

			lp.coordinator = coordinator.NewAdapter(lp, coordinator.New(util.NewLogger("foo"), []api.Vehicle{vehicle}))

			attachListeners(t, lp)

			// mode now
			charger.EXPECT().MaxCurrent(int64(maxA))
			// sync charger
			charger.EXPECT().Enabled().Return(true, nil)

			// vehicle not updated yet
			vehicle.MockChargeState.EXPECT().Status().Return(api.StatusA, nil)

			lp.Update(0, false, false)
			ctrl.Finish()

			// detection started
			assert.Equal(t, lp.clock.Now(), lp.vehicleDetect, "vehicle detection not started")

			// vehicle not detected yet
			assert.Nil(t, lp.vehicle, "vehicle should be <nil>")

			// sync charger
			charger.EXPECT().Enabled().Return(true, nil)
			// vehicle not updated yet
			vehicle.MockChargeState.EXPECT().Status().Return(api.StatusB, nil)

			lp.Update(0, false, false)
			ctrl.Finish()

			// vehicle detected
			assert.Equal(t, vehicle, lp.vehicle, "vehicle should be detected")
		})
	}
}
