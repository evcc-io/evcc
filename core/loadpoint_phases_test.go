package core

import (
	"errors"
	"strings"
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type testCase struct {
	// capable=0 signals 1p3p as set during loadpoint init
	// physical/vehicle=0 signals unknown
	// measuredPhases<>0 signals previous measurement
	capable, physical, vehicle, measuredPhases, actExpected, maxExpected, minExpected int
	// scaling expectation: d=down, u=up, du=both
	scale string
}

var phaseTests = []testCase{
	// 1p
	{1, 1, 0, 0, 1, 1, 1, ""},
	{1, 1, 0, 1, 1, 1, 1, ""},
	{1, 1, 1, 0, 1, 1, 1, ""},
	{1, 1, 2, 0, 1, 1, 1, ""},
	{1, 1, 3, 0, 1, 1, 1, ""},
	// 3p
	{3, 3, 0, 0, unknownPhases, 3, 3, ""},
	{3, 3, 0, 1, 1, 1, 1, ""},
	{3, 3, 0, 2, 2, 2, 2, ""},
	{3, 3, 0, 3, 3, 3, 3, ""},
	{3, 3, 1, 0, 1, 1, 1, ""},
	{3, 3, 2, 0, 2, 2, 2, ""},
	{3, 3, 3, 0, 3, 3, 3, ""},
	// 1p3p initial
	{0, 0, 0, 0, unknownPhases, 3, 1, "du"},
	{0, 0, 0, 1, 1, 3, 1, "u"},
	{0, 0, 0, 2, 2, 3, 1, "du"},
	{0, 0, 0, 3, 3, 3, 1, "du"},
	{0, 0, 1, 0, 1, 1, 1, ""},
	{0, 0, 2, 0, 2, 2, 1, "du"},
	{0, 0, 3, 0, 3, 3, 1, "du"},
	// 1p3p, 1 currently active
	{0, 1, 0, 0, 1, 3, 1, "u"},
	{0, 1, 0, 1, 1, 3, 1, "u"},
	// {0, 1, 0, 2, 2,2,"u"}, // 2p active > 1p configured must not happen
	// {0, 1, 0, 3, 3,3,"u"}, // 3p active > 1p configured must not happen
	{0, 1, 1, 0, 1, 1, 1, ""},
	{0, 1, 2, 0, 1, 2, 1, "u"},
	{0, 1, 3, 0, 1, 3, 1, "u"},
	// 1p3p, 3 currently active
	{0, 3, 0, 0, unknownPhases, 3, 1, "d"},
	{0, 3, 0, 1, 1, 1, 1, ""},
	{0, 3, 0, 2, 2, 2, 1, "d"},
	{0, 3, 0, 3, 3, 3, 1, "d"},
	{0, 3, 1, 0, 1, 1, 1, ""},
	{0, 3, 2, 0, 2, 2, 1, "d"},
	{0, 3, 3, 0, 3, 3, 1, "d"},
}

func TestMaxActivePhases(t *testing.T) {
	// 0 is auto, 1/3 are fixed
	for _, configured := range []int{0, 1, 3} {
		for _, tc := range phaseTests {
			// skip invalid configs (free scaling for simple charger)
			if configured == 0 && tc.capable != 0 {
				continue
			}

			t.Logf("configured %d %+v", configured, tc)

			ctrl := gomock.NewController(t)
			plainCharger := api.NewMockCharger(ctrl)

			vehicle := api.NewMockVehicle(ctrl)
			vehicle.EXPECT().Phases().Return(tc.vehicle).MinTimes(1)

			lp := &Loadpoint{
				phasesConfigured: configured, // fixed phases or default
				vehicle:          vehicle,
				phases:           tc.physical,
				measuredPhases:   tc.measuredPhases,
				charger:          plainCharger,
			}

			// 1p3p
			if tc.capable == 0 {
				lp.charger = struct {
					*api.MockCharger
					*api.MockPhaseSwitcher
				}{
					plainCharger, api.NewMockPhaseSwitcher(ctrl),
				}
			}

			// restrict scalable charger by config
			expectedPhases := tc.maxExpected
			if tc.capable == 0 && configured > 0 && configured < tc.maxExpected {
				expectedPhases = configured
			}

			require.Equal(t, expectedPhases, lp.maxActivePhases(), "expected max active phases")
			ctrl.Finish()
		}
	}
}

func TestMinActivePhases(t *testing.T) {
	// 0 is auto, 1/3 are fixed
	for _, configured := range []int{0, 1, 3} {
		for _, tc := range phaseTests {
			// skip invalid configs (free scaling for simple charger)
			if configured == 0 && tc.capable != 0 {
				continue
			}

			// skip physical config different than configured
			if configured != 0 && tc.capable != 0 && configured != tc.physical {
				continue
			}

			t.Logf("configured %d %+v", configured, tc)

			ctrl := gomock.NewController(t)
			plainCharger := api.NewMockCharger(ctrl)

			vehicle := api.NewMockVehicle(ctrl)
			vehicle.EXPECT().Phases().Return(tc.vehicle).AnyTimes()

			lp := &Loadpoint{
				phasesConfigured: configured, // fixed phases or default
				vehicle:          vehicle,
				phases:           tc.physical,
				measuredPhases:   tc.measuredPhases,
				charger:          plainCharger,
			}

			// 1p3p
			if tc.capable == 0 {
				lp.charger = struct {
					*api.MockCharger
					*api.MockPhaseSwitcher
				}{
					plainCharger, api.NewMockPhaseSwitcher(ctrl),
				}
			}

			// restrict scalable charger by config
			expectedPhases := tc.minExpected
			if tc.capable == 0 && configured > 0 && configured < tc.minExpected {
				expectedPhases = configured
			}

			require.Equal(t, expectedPhases, lp.minActivePhases(), "expected min active phases")
			ctrl.Finish()
		}
	}
}

func testScale(t *testing.T, lp *Loadpoint, sitePower float64, direction string, tc testCase) {
	t.Helper()

	act := lp.ActivePhases()
	max := lp.maxActivePhases()

	testDirection := direction[0:1] // (d)own or (u)p
	testExpectation := tc.scale

	// up-scale expected
	if testDirection == "u" && strings.Contains(testExpectation, testDirection) {
		// scale-up should only execute when the 1p max current is exceeded
		// we're testing this here and remove the upscale expectation for the following test below 1p max current
		if maxAmp := -sitePower / Voltage; maxAmp < maxA {
			if scaled := lp.pvScalePhases(sitePower, minA, maxAmp-0.0001); scaled != 3 {
				t.Errorf("%v act=%d max=%d missing scale %s at reduced max current %.1fA", tc, act, max, direction, maxAmp)
			}

			// we've verified scale-up here so next test should not scale
			testExpectation = strings.ReplaceAll(testExpectation, testDirection, "")
		}
	}

	scaled := lp.pvScalePhases(sitePower, minA, maxA)

	if strings.Contains(testExpectation, testDirection) {
		if scaled == 0 {
			t.Errorf("%v act=%d max=%d missing scale %s", tc, act, max, direction)
		}
	} else if scaled != 0 {
		t.Errorf("%v act=%d max=%d unexpected scale %s", tc, act, max, direction)
	}
}

func TestPvScalePhases(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)

	for _, tc := range phaseTests {
		t.Log(tc)

		plainCharger := api.NewMockCharger(ctrl)
		plainCharger.EXPECT().Enabled().Return(true, nil)
		plainCharger.EXPECT().MaxCurrent(int64(minA)).Return(nil) // MaxCurrentEx not implemented

		// 1p3p
		var phaseCharger *api.MockPhaseSwitcher
		if tc.capable == 0 {
			phaseCharger = api.NewMockPhaseSwitcher(ctrl)
		}

		vehicle := api.NewMockVehicle(ctrl)
		vehicle.EXPECT().Phases().Return(tc.vehicle).MinTimes(1)
		vehicle.EXPECT().OnIdentified().Return(api.ActionConfig{}).AnyTimes()

		lp := &Loadpoint{
			log:              util.NewLogger("foo"),
			bus:              evbus.New(),
			clock:            clock,
			chargeMeter:      &Null{},            // silence nil panics
			chargeRater:      &Null{},            // silence nil panics
			chargeTimer:      &Null{},            // silence nil panics
			progress:         NewProgress(0, 10), // silence nil panics
			wakeUpTimer:      NewTimer(),         // silence nil panics
			mode:             api.ModeNow,
			minCurrent:       minA,
			maxCurrent:       maxA,
			vehicle:          vehicle,
			phasesConfigured: 0, // allow switching
			phases:           tc.physical,
			status:           api.StatusC,
		}

		if phaseCharger != nil {
			lp.charger = struct {
				*api.MockCharger
				*api.MockPhaseSwitcher
			}{
				plainCharger, phaseCharger,
			}
		} else {
			lp.charger = struct {
				*api.MockCharger
			}{
				plainCharger,
			}
		}

		attachListeners(t, lp)

		lp.measuredPhases = tc.measuredPhases
		if tc.measuredPhases > 0 && tc.vehicle > 0 {
			t.Fatalf("%v invalid test case", tc)
		}

		require.Equal(t, tc.physical, lp.phases, "wrong phases")
		require.Equal(t, tc.actExpected, lp.ActivePhases(), "expected active phases")
		require.Equal(t, tc.maxExpected, lp.maxActivePhases(), "expected max active phases")

		ctrl.Finish()

		// scaling
		if phaseCharger != nil {
			// scale down
			min1p := 0.1
			lp.phaseTimer = time.Time{}

			plainCharger.EXPECT().Enable(false).Return(nil).MaxTimes(1)
			phaseCharger.EXPECT().Phases1p3p(1).Return(nil).MaxTimes(1)

			testScale(t, lp, min1p, "down", tc)
			ctrl.Finish()

			// scale up
			min3p := float64(tc.maxExpected) * minA * Voltage
			lp.phaseTimer = time.Time{}

			// reset to initial state
			lp.phases = tc.physical
			lp.measuredPhases = tc.measuredPhases

			plainCharger.EXPECT().Enable(false).Return(nil).MaxTimes(1)
			phaseCharger.EXPECT().Phases1p3p(3).Return(nil).MaxTimes(1)

			testScale(t, lp, -min3p, "up", tc)
			ctrl.Finish()
		}
	}
}

func TestPvScalePhasesTimer(t *testing.T) {
	ctrl := gomock.NewController(t)
	charger := &struct {
		*api.MockCharger
		*api.MockPhaseSwitcher
	}{
		api.NewMockCharger(ctrl),
		api.NewMockPhaseSwitcher(ctrl),
	}

	dt := time.Minute
	Voltage = 230 // V

	tc := []struct {
		desc                   string
		phases, measuredPhases int
		sitePower              float64
		toPhases               int
		res                    int
		prepare                func(lp *Loadpoint)
	}{
		// switch up from 1p/1p configured/active
		{"1/1->3, not enough power", 1, 1, 0, 1, 0, nil},
		{"1/1->3, kickoff", 1, 1, -3 * Voltage * minA, 1, 0, func(lp *Loadpoint) {
			lp.phaseTimer = time.Time{}
		}},
		{"1/1->3, timer running", 1, 1, -3 * Voltage * minA, 1, 0, func(lp *Loadpoint) {
			lp.phaseTimer = lp.clock.Now()
		}},
		{"1/1->3, timer elapsed", 1, 1, -3 * Voltage * minA, 3, 3, func(lp *Loadpoint) {
			lp.phaseTimer = elapsed
		}},

		// omit to switch up (again) from 3p/1p configured/a0ctive
		{"3/1->3, not enough power", 3, 1, 0, 3, 0, nil},
		{"3/1->3, kickoff", 3, 1, -3 * Voltage * minA, 3, 0, func(lp *Loadpoint) {
			lp.phaseTimer = time.Time{}
		}},
		{"3/1->3, timer running", 3, 1, -3 * Voltage * minA, 3, 0, func(lp *Loadpoint) {
			lp.phaseTimer = lp.clock.Now()
		}},
		{"3/1->3, timer elapsed", 3, 1, -3 * Voltage * minA, 3, 0, func(lp *Loadpoint) {
			lp.phaseTimer = elapsed
		}},

		// omit to switch down from 3p/1p configured/active
		{"3/1->1, not enough power", 3, 1, 0, 3, 0, nil},
		{"3/1->1, kickoff", 3, 1, -1 * Voltage * minA, 3, 0, func(lp *Loadpoint) {
			lp.phaseTimer = time.Time{}
		}},
		{"3/1->1, timer running", 3, 1, -1 * Voltage * minA, 3, 0, func(lp *Loadpoint) {
			lp.phaseTimer = lp.clock.Now()
		}},
		{"3/1->1, timer elapsed", 3, 1, -1 * Voltage * minA, 3, 0, func(lp *Loadpoint) {
			lp.phaseTimer = elapsed
		}},

		// switch down from 3p/3p configured/active
		{"3/3->1, enough power", 3, 3, 0, 3, 0, nil},
		{"3/3->1, enough power, timer elapsed, load point enabled", 3, 3, 0, 3, 0, func(lp *Loadpoint) {
			lp.phaseTimer = elapsed
			lp.enabled = true
		}},
		{"3/3->1, enough power, timer elapsed, load point disabled", 3, 3, 0, 1, 1, func(lp *Loadpoint) {
			lp.phaseTimer = elapsed
			lp.enabled = false
		}},
		{"3/3->1, kickoff", 3, 3, 0.1, 3, 0, func(lp *Loadpoint) {
			lp.phaseTimer = time.Time{}
		}},
		{"3/3->1, timer running", 3, 3, 0.1, 3, 0, func(lp *Loadpoint) {
			lp.phaseTimer = lp.clock.Now()
		}},
		{"3/3->1, timer elapsed", 3, 3, 0.1, 1, 1, func(lp *Loadpoint) {
			lp.phaseTimer = elapsed
		}},

		// switch down from 3p/0p while not yet charging
		{"3/0->1, not enough power, not charging", 3, 0, 0, 1, 1, func(lp *Loadpoint) {
			lp.status = api.StatusB
		}},
		// switch up from 1p/0p while not yet charging
		{"1/0->3, enough power, not charging", 1, 0, -3 * Voltage * minA, 3, 3, func(lp *Loadpoint) {
			lp.status = api.StatusB
		}},

		// error states from 1p/3p misconfiguration - no correction for time being (stay at 1p)
		{"1/3->1, enough power", 1, 3, -1 * Voltage * maxA, 1, 0, nil},
		{"1/3->1, kickoff, correct phase setting", 1, 3, 0.1, 1, 0, func(lp *Loadpoint) {
			lp.phaseTimer = time.Time{}
		}},
		{"1/3->1, timer running, correct phase setting", 1, 3, 0.1, 1, 0, func(lp *Loadpoint) {
			lp.phaseTimer = lp.clock.Now()
		}},
		{"1/3->1, switch not executed", 1, 3, 0.1, 1, 0, func(lp *Loadpoint) {
			lp.phaseTimer = elapsed
		}},
	}

	for _, tc := range tc {
		t.Logf("%+v", tc)
		clock := clock.NewMock()
		clock.Add(time.Hour) // avoid time.IsZero

		lp := &Loadpoint{
			log:            util.NewLogger("foo"),
			wakeUpTimer:    NewTimer(),
			clock:          clock,
			charger:        charger,
			minCurrent:     minA,
			maxCurrent:     maxA,
			phases:         tc.phases,
			measuredPhases: tc.measuredPhases,
			status:         api.StatusC,
			Enable: loadpoint.ThresholdConfig{
				Delay: dt,
			},
			Disable: loadpoint.ThresholdConfig{
				Delay: dt,
			},
		}

		if tc.prepare != nil {
			tc.prepare(lp)
		}

		if tc.res != 0 {
			charger.MockPhaseSwitcher.EXPECT().Phases1p3p(tc.toPhases).Return(nil)
		}

		res := lp.pvScalePhases(tc.sitePower, minA, maxA)

		require.Equal(t, tc.res, res, tc.desc)
		require.Equal(t, tc.toPhases, lp.phases, tc.desc)
	}
}

func TestScalePhasesIfAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)

	tc := []struct {
		dflt, physical, maxExpected int
	}{
		{0, 0, 3},
		{0, 1, 3},
		{0, 3, 3},
		{1, 0, 1},
		{1, 1, 1},
		{1, 3, 1},
		{3, 0, 3},
		{3, 1, 3},
		{3, 3, 3},
	}

	for _, tc := range tc {
		t.Log(tc)

		plainCharger := api.NewMockCharger(ctrl)
		phaseCharger := api.NewMockPhaseSwitcher(ctrl)

		lp := &Loadpoint{
			log:         util.NewLogger("foo"),
			clock:       clock.NewMock(),
			wakeUpTimer: NewTimer(),
			charger: struct {
				*api.MockCharger
				*api.MockPhaseSwitcher
			}{
				plainCharger,
				phaseCharger,
			},
			minCurrent:       minA,
			phasesConfigured: tc.dflt,     // fixed phases or default
			phases:           tc.physical, // current phase status
		}

		// restrict scalable charger by config
		if tc.dflt == 0 || tc.dflt != tc.physical {
			phaseCharger.EXPECT().Phases1p3p(tc.maxExpected).Return(nil)
		}

		_ = lp.scalePhasesIfAvailable(3)

		ctrl.Finish()
	}
}

// TestScalePhasesNotAvailable verifies that a charger reporting api.ErrNotAvailable
// from Phases1p3p (e.g. an EEBus charger with an ISO 15118 vehicle, see issue #29974)
// is treated as a failed switch: the error is api.ErrNotAvailable so callers can
// suppress it, and the phase count is left unchanged.
func TestScalePhasesNotAvailable(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	plainCharger := api.NewMockCharger(ctrl)
	phaseCharger := api.NewMockPhaseSwitcher(ctrl)
	phaseCharger.EXPECT().Phases1p3p(3).Return(api.ErrNotAvailable)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		clock:       clock.NewMock(),
		wakeUpTimer: NewTimer(),
		charger: struct {
			*api.MockCharger
			*api.MockPhaseSwitcher
		}{
			plainCharger,
			phaseCharger,
		},
		phases: 1, // current phase status, switch to 3p will be attempted
	}

	err := lp.scalePhases(3)
	require.Error(t, err)
	require.True(t, errors.Is(err, api.ErrNotAvailable), "want api.ErrNotAvailable, got %v", err)

	// switch did not complete - phase count unchanged
	require.Equal(t, 1, lp.GetPhases())
}

func TestFastChargingCircuitBasedPhaseScaling(t *testing.T) {
	Voltage = 230

	tc := []struct {
		desc                  string
		phases                int
		chargePower           float64
		availableCircuitPower float64 // ValidatePower return for 3p request
		expectedPhases        int
		noCircuit             bool
	}{
		{desc: "no circuit", phases: 3, chargePower: 0, expectedPhases: 3, noCircuit: true},
		{desc: "low limit, no surplus", phases: 3, chargePower: 0, availableCircuitPower: 3680, expectedPhases: 1},
		{desc: "low limit, with surplus", phases: 1, chargePower: 0, availableCircuitPower: 11040, expectedPhases: 3},
		{desc: "already charging, low limit", phases: 3, chargePower: 3680, availableCircuitPower: 3680, expectedPhases: 1},
		{desc: "already charging, high limit", phases: 1, chargePower: 3680, availableCircuitPower: 11040, expectedPhases: 3},
		{desc: "edge case: just below 3p minimum", phases: 3, chargePower: 0, availableCircuitPower: 4140 - 1, expectedPhases: 1},
		{desc: "edge case: just at 3p minimum", phases: 1, chargePower: 0, availableCircuitPower: 4140, expectedPhases: 3},
	}

	for _, tc := range tc {
		t.Run(tc.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			lp := NewLoadpoint(util.NewLogger("foo"), nil)
			lp.minCurrent = 6
			lp.maxCurrent = 16
			lp.phases = tc.phases
			lp.chargePower = tc.chargePower
			lp.offeredCurrent = 0 // ensure MaxCurrent is called
			lp.wakeUpTimer = NewTimer()

			plainCharger := api.NewMockCharger(ctrl)
			phaseCharger := api.NewMockPhaseSwitcher(ctrl)
			lp.charger = struct {
				*api.MockCharger
				*api.MockPhaseSwitcher
			}{plainCharger, phaseCharger}

			if !tc.noCircuit {
				circuit := api.NewMockCircuit(ctrl)
				lp.circuit = circuit

				minPower3p := Voltage * 6 * 3

				// fastCharging call to ValidatePower
				circuit.EXPECT().ValidatePower(tc.chargePower, minPower3p).Return(tc.availableCircuitPower)

				// setLimit calls
				circuit.EXPECT().ValidateCurrent(gomock.Any(), lp.maxCurrent).Return(lp.maxCurrent)
				circuit.EXPECT().ValidatePower(tc.chargePower, float64(tc.expectedPhases)*Voltage*lp.maxCurrent).Return(float64(tc.expectedPhases) * Voltage * lp.maxCurrent)
			}

			plainCharger.EXPECT().Enabled().Return(true, nil).AnyTimes()
			plainCharger.EXPECT().Enable(gomock.Any()).Return(nil).AnyTimes()

			if tc.phases != tc.expectedPhases {
				phaseCharger.EXPECT().Phases1p3p(tc.expectedPhases).Return(nil)
			}

			plainCharger.EXPECT().MaxCurrent(int64(lp.maxCurrent)).Return(nil)

			err := lp.fastCharging()
			require.NoError(t, err)
			require.Equal(t, tc.expectedPhases, lp.phases, tc.desc)

			ctrl.Finish()
		})
	}
}

// TestUpdatePhaseSwitchNotAvailable verifies that, with a fixed phasesConfigured
// and a charger that refuses phase switching (api.ErrNotAvailable, e.g. an EEBus
// charger with an ISO 15118 vehicle), the configured phase count is adopted so
// the switch is not re-attempted on every update cycle (issue #29974).
func TestUpdatePhaseSwitchNotAvailable(t *testing.T) {
	clock := clock.NewMock()
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	plainCharger := api.NewMockCharger(ctrl)
	phaseCharger := api.NewMockPhaseSwitcher(ctrl)

	plainCharger.EXPECT().Status().Return(api.StatusC, nil).AnyTimes()
	plainCharger.EXPECT().Enabled().Return(true, nil).AnyTimes()
	plainCharger.EXPECT().Enable(gomock.Any()).Return(nil).AnyTimes()
	plainCharger.EXPECT().MaxCurrent(gomock.Any()).Return(nil).AnyTimes()

	// charger cannot switch phases - and must only be asked once
	phaseCharger.EXPECT().Phases1p3p(3).Return(api.ErrNotAvailable).Times(1)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		chargeMeter: &Null{},
		chargeRater: &Null{},
		chargeTimer: &Null{},
		progress:    NewProgress(0, 10),
		wakeUpTimer: NewTimer(),
		mode:        api.ModeNow,
		minCurrent:  minA,
		maxCurrent:  maxA,
		status:      api.StatusC,
		charger: struct {
			*api.MockCharger
			*api.MockPhaseSwitcher
		}{
			plainCharger,
			phaseCharger,
		},
		phasesConfigured: 3, // fixed 3p
		phases:           0, // unknown
	}

	attachListeners(t, lp)

	// first cycle attempts the switch, gets api.ErrNotAvailable, adopts 3p
	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil)
	require.Equal(t, 3, lp.GetPhases(), "configured phases should be adopted")

	// second cycle must not attempt the switch again (Phases1p3p .Times(1))
	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil)
	require.Equal(t, 3, lp.GetPhases())
}
