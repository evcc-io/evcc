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

func TestPublishSoCAndRange(t *testing.T) {
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
	lp := &LoadPoint{
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

		assert.True(t, lp.socPollAllowed())
		vehicle.EXPECT().SoC().Return(0.0, errors.New("foo"))
		lp.publishSoCAndRange()

		clck.Add(time.Second)
		assert.Equal(t, tc.allowed, lp.socPollAllowed())
		if tc.allowed {
			vehicle.EXPECT().SoC().Return(0.0, errors.New("foo"))
		}
		lp.publishSoCAndRange()
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

		lp := &LoadPoint{
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
		MinSoC:    &minsoc,
		TargetSoC: &targetsoc,
	}).AnyTimes()

	vehicle := mock.NewMockVehicle(ctrl)
	vehicle.EXPECT().Title().Return("target").AnyTimes()
	vehicle.EXPECT().Icon().Return("").AnyTimes()
	vehicle.EXPECT().Capacity().AnyTimes()
	vehicle.EXPECT().Phases().AnyTimes()
	vehicle.EXPECT().OnIdentified().AnyTimes()

	lp := NewLoadPoint(util.NewLogger("foo"))
	lp.defaultVehicle = dflt
	lp.collectDefaults()

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
	if lp.vehicle != vehicle {
		t.Errorf("expected %v, got %v", title(vehicle), title(lp.vehicle))
	}

	// non-default vehicle disconnected
	lp.evVehicleDisconnectHandler()
	if lp.vehicle != dflt {
		t.Errorf("expected %v, got %v", title(dflt), title(lp.vehicle))
	}

	// default vehicle disconnected
	lp.ResetOnDisconnect = true
	lp.evVehicleDisconnectHandler()
	if m := lp.GetMode(); m != mode {
		t.Errorf("expected mode %v, got %v", mode, m)
	}
	if s := lp.GetMinSoC(); s != minsoc {
		t.Errorf("expected minsoc %v, got %v", minsoc, s)
	}
	if s := lp.GetTargetSoC(); s != targetsoc {
		t.Errorf("expected targetsoc %v, got %v", targetsoc, s)
	}
	if m := lp.onDisconnect.Mode; *m != api.ModeOff {
		t.Errorf("expected ondisconnect mode %v, got %v", api.ModeOff, m)
	}

	// set non-default vehicle during disconnect - should be default on connect
	lp.tasks.Clear()
	lp.evVehicleConnectHandler()
	if lp.vehicle != dflt {
		t.Errorf("expected %v, got %v", title(dflt), title(lp.vehicle))
	}
	if l := lp.tasks.Size(); l != 1 {
		t.Error("expected task in queue, got none")
	}

	// guest connected
	lp.setActiveVehicle(nil)
	if lp.vehicle != nil {
		t.Errorf("expected %v, got %v", nil, title(lp.vehicle))
	}
}

func TestApplyVehicleDefaults(t *testing.T) {
	ctrl := gomock.NewController(t)

	newConfig := func(mode api.ChargeMode, minCurrent, maxCurrent float64, minSoC, targetSoC int) api.ActionConfig {
		return api.ActionConfig{
			Mode:       &mode,
			MinCurrent: &minCurrent,
			MaxCurrent: &maxCurrent,
			MinSoC:     &minSoC,
			TargetSoC:  &targetSoC,
		}
	}

	assertConfig := func(lp *LoadPoint, conf api.ActionConfig) {
		if lp.Mode != *conf.Mode {
			t.Errorf("expected mode %v, got %v", *conf.Mode, lp.Mode)
		}
		if lp.MinCurrent != *conf.MinCurrent {
			t.Errorf("expected minCurrent %v, got %v", *conf.MinCurrent, lp.MinCurrent)
		}
		if lp.MaxCurrent != *conf.MaxCurrent {
			t.Errorf("expected maxCurrent %v, got %v", *conf.MaxCurrent, lp.MaxCurrent)
		}
		if lp.SoC.min != *conf.MinSoC {
			t.Errorf("expected minSoC %v, got %v", *conf.MinSoC, lp.SoC.min)
		}
		if lp.SoC.target != *conf.TargetSoC {
			t.Errorf("expected targetSoC %v, got %v", *conf.TargetSoC, lp.SoC.target)
		}
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

	lp := NewLoadPoint(util.NewLogger("foo"))

	// populate channels
	x, y, z := createChannels(t)
	attachChannels(lp, x, y, z)

	lp.onDisconnect = od
	lp.ResetOnDisconnect = true

	// check loadpoint default currents can't be violated
	lp.applyAction(newConfig(*od.Mode, 5, 17, *od.MinSoC, *od.TargetSoC))
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
			vehicle.MockVehicle.EXPECT().SoC().Return(0.0, nil).AnyTimes()

			charger := mock.NewMockCharger(ctrl)
			charger.EXPECT().Status().Return(api.StatusB, nil).AnyTimes()

			lp := &LoadPoint{
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
			if lp.vehicleDetect != lp.clock.Now() {
				t.Error("vehicle detection not started")
			}

			// vehicle not detected yet
			if lp.vehicle != nil {
				t.Error("vehicle should be <nil>")
			}

			// sync charger
			charger.EXPECT().Enabled().Return(true, nil)
			// vehicle not updated yet
			vehicle.MockChargeState.EXPECT().Status().Return(api.StatusB, nil)

			lp.Update(0, false, false)
			ctrl.Finish()

			// vehicle detected
			if lp.vehicle != vehicle {
				t.Error("vehicle should be detected")
			}
		})
	}
}
