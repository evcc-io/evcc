package core

// 0A safety sequence for phase switching.
//
// Switching phases under load can damage the installation contactor (Shelly
// Pro 1PM Gen3) and the vehicle's charging electronics, so the load is
// dropped to 0A first and only then is the physical switch executed. The
// safety buffer (phaseSwitchSafetyBuffer = 20s) is spent at confirmed zero
// current: the timer is anchored on the first tick with the charger idle
// (status != C) and the measured current below minActiveCurrent, and it
// restarts whenever current reappears or the charger reports charging
// again. While the hold runs, the 0A limit is re-applied (rate limited) so
// a vehicle that starts drawing again is stopped. After the switch the pv
// logic re-establishes the charge current from the current surplus.
//
// The tests below simulate the mode dispatch (fastCharging = "Schnell"/now,
// minCharging = minpv) and the advancePhaseSwitch tick that runs every cycle
// while the sequence is pending.

import (
	"testing"
	"time"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// TestPhaseSwitchSafetySequenceFastCharging simulates the "Schnell" (now)
// mode: a 1p→3p switch is requested while the vehicle is charging at max
// current. The load must be dropped to 0A, the safety buffer waited out,
// only then the switch happens, and max current is restored afterwards.
func TestPhaseSwitchSafetySequenceFastCharging(t *testing.T) {
	ctrl := gomock.NewController(t)

	// 1) 0A drop, 2) the physical switch after the buffer, 3) max current again
	chargerMock := api.NewMockCharger(ctrl)
	chargerMock.EXPECT().Enable(false).Return(nil)
	chargerMock.EXPECT().MaxCurrent(int64(18)).Return(nil)
	chargerMock.EXPECT().Enable(true).Return(nil)
	phaseMock := api.NewMockPhaseSwitcher(ctrl)
	phaseMock.EXPECT().Phases1p3p(3).Return(nil).Times(1)

	clock := clock.NewMock()
	clock.Add(time.Hour)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		wakeUpTimer: NewTimer(),
		charger: struct {
			*api.MockCharger
			*api.MockPhaseSwitcher
		}{chargerMock, phaseMock},
		minCurrent:     2,
		maxCurrent:     18,
		phases:         1,
		status:         api.StatusC, // charging at 18A
		enabled:        true,
		offeredCurrent: 18,
	}

	// "Schnell": request 3p while charging → sequence starts, load drops to 0A
	require.NoError(t, lp.fastCharging())
	require.Equal(t, 3, lp.phaseSwitchTarget)
	require.Equal(t, 1, lp.phases)
	require.False(t, lp.enabled)
	require.Equal(t, 0.0, lp.offeredCurrent)

	// inside the buffer: no switch
	clock.Add(10 * time.Second)
	require.NoError(t, lp.advancePhaseSwitch())
	require.Equal(t, 1, lp.phases)
	require.Equal(t, 3, lp.phaseSwitchTarget)
	require.True(t, lp.phaseSwitchZeroAt.IsZero())

	// buffer elapsed since the request, but the charger still reports
	// charging → no anchor yet, so still no switch under load
	clock.Add(11 * time.Second)
	require.NoError(t, lp.advancePhaseSwitch())
	require.Equal(t, 1, lp.phases)
	require.True(t, lp.phaseSwitchZeroAt.IsZero())

	// charger idle now (status B: vehicle connected, contactor open) →
	// the 0A anchor is set on this tick
	lp.status = api.StatusB
	require.NoError(t, lp.advancePhaseSwitch())
	require.Equal(t, 1, lp.phases)
	require.False(t, lp.phaseSwitchZeroAt.IsZero())

	// safety buffer elapsed since the anchor → switch
	clock.Add(21 * time.Second)
	require.NoError(t, lp.advancePhaseSwitch())
	require.Equal(t, 3, lp.phases)
	require.Equal(t, 0, lp.phaseSwitchTarget)

	// next cycle: max current is restored
	require.NoError(t, lp.fastCharging())
	require.Equal(t, 18.0, lp.offeredCurrent)
	require.True(t, lp.enabled)
}

// TestPhaseSwitchSafetySequenceMinCharging simulates minpv: a 3p→1p switch is
// requested while the vehicle is charging at min current.
func TestPhaseSwitchSafetySequenceMinCharging(t *testing.T) {
	ctrl := gomock.NewController(t)

	chargerMock := api.NewMockCharger(ctrl)
	chargerMock.EXPECT().Enable(false).Return(nil)
	chargerMock.EXPECT().MaxCurrent(int64(2)).Return(nil)
	chargerMock.EXPECT().Enable(true).Return(nil)
	phaseMock := api.NewMockPhaseSwitcher(ctrl)
	phaseMock.EXPECT().Phases1p3p(1).Return(nil).Times(1)

	clock := clock.NewMock()
	clock.Add(time.Hour)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		wakeUpTimer: NewTimer(),
		charger: struct {
			*api.MockCharger
			*api.MockPhaseSwitcher
		}{chargerMock, phaseMock},
		minCurrent:     2,
		maxCurrent:     18,
		phases:         3,
		status:         api.StatusC, // charging at 2A
		enabled:        true,
		offeredCurrent: 2,
	}

	// minpv: request 1p while charging → sequence starts, load drops to 0A
	require.NoError(t, lp.minCharging())
	require.Equal(t, 1, lp.phaseSwitchTarget)
	require.Equal(t, 3, lp.phases)
	require.False(t, lp.enabled)
	require.Equal(t, 0.0, lp.offeredCurrent)

	// charger idle (status B) → the 0A anchor is set on this tick
	clock.Add(21 * time.Second)
	lp.status = api.StatusB
	require.NoError(t, lp.advancePhaseSwitch())
	require.Equal(t, 3, lp.phases)
	require.False(t, lp.phaseSwitchZeroAt.IsZero())

	// safety buffer elapsed since the anchor → switch
	clock.Add(21 * time.Second)
	require.NoError(t, lp.advancePhaseSwitch())
	require.Equal(t, 1, lp.phases)
	require.Equal(t, 0, lp.phaseSwitchTarget)

	// next cycle: min current is restored
	require.NoError(t, lp.minCharging())
	require.Equal(t, 2.0, lp.offeredCurrent)
	require.True(t, lp.enabled)
}

// TestPhaseSwitchAbortOnDisconnect verifies that a pending sequence is aborted
// (and never switched) when the vehicle disconnects.
func TestPhaseSwitchAbortOnDisconnect(t *testing.T) {
	ctrl := gomock.NewController(t)

	chargerMock := api.NewMockCharger(ctrl)
	chargerMock.EXPECT().Enable(false).Return(nil)
	phaseMock := api.NewMockPhaseSwitcher(ctrl)
	// the switch must never happen
	phaseMock.EXPECT().Phases1p3p(gomock.Any()).Return(nil).Times(0)

	clock := clock.NewMock()
	clock.Add(time.Hour)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		wakeUpTimer: NewTimer(),
		charger: struct {
			*api.MockCharger
			*api.MockPhaseSwitcher
		}{chargerMock, phaseMock},
		minCurrent:     2,
		maxCurrent:     18,
		phases:         3,
		status:         api.StatusC,
		enabled:        true,
		offeredCurrent: 16,
	}

	// start the 3p→1p sequence
	require.NoError(t, lp.minCharging())
	require.Equal(t, 1, lp.phaseSwitchTarget)

	// vehicle unplugs (status A) → the dispatch keeps 0A and cancels the switch
	lp.status = api.StatusA
	require.NoError(t, lp.setLimit(0))
	lp.cancelPhaseSwitch()
	require.Equal(t, 0, lp.phaseSwitchTarget)

	// even after the buffer elapsed there is nothing to execute
	clock.Add(time.Minute)
	require.NoError(t, lp.advancePhaseSwitch())
	require.Equal(t, 3, lp.phases)
}

// TestPhaseSwitchAbortOnDisconnectViaUpdate verifies that the abort also
// works through the real update dispatch: a vehicle that unplugs while the
// 0A safety sequence is pending must keep 0A, cancel the pending switch and
// never execute the physical phase change.
func TestPhaseSwitchAbortOnDisconnectViaUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	plainCharger := api.NewMockCharger(ctrl)
	phaseCharger := api.NewMockPhaseSwitcher(ctrl)

	chargerStatus := api.StatusC
	plainCharger.EXPECT().Status().DoAndReturn(func() (api.ChargeStatus, error) {
		return chargerStatus, nil
	}).AnyTimes()
	plainCharger.EXPECT().Enabled().Return(true, nil).AnyTimes()
	plainCharger.EXPECT().Enable(gomock.Any()).Return(nil).AnyTimes()
	plainCharger.EXPECT().MaxCurrent(gomock.Any()).Return(nil).AnyTimes()
	// the physical switch must never happen
	phaseCharger.EXPECT().Phases1p3p(gomock.Any()).Return(nil).Times(0)

	clock := clock.NewMock()

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		chargeMeter: newChargeMeter(&Null{}),
		chargeRater: &Null{},
		chargeTimer: &Null{},
		progress:    NewProgress(0, 10),
		wakeUpTimer: NewTimer(),
		mode:        api.ModePV,
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
		phases:           3,
		phasesConfigured: 0,
		enabled:          true,
		offeredCurrent:   16,
	}

	attachListeners(t, lp)

	// a 3p->1p sequence is pending (started by pvScalePhases while charging)
	require.NoError(t, lp.scalePhases(1))
	require.Equal(t, 1, lp.phaseSwitchTarget)
	require.False(t, lp.enabled)

	// vehicle still connected: the update dispatch keeps the hold at 0A
	clock.Add(10 * time.Second)
	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, nil)
	require.True(t, lp.phaseSwitchPending())
	require.False(t, lp.enabled)
	require.Equal(t, 0.0, lp.offeredCurrent)

	// vehicle unplugs: the dispatch cancels the pending switch and stays at 0A
	chargerStatus = api.StatusA
	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, nil)
	require.Equal(t, 0, lp.phaseSwitchTarget, "disconnect must cancel the pending switch")
	require.Equal(t, 3, lp.GetPhases())
	require.False(t, lp.phaseSwitchPending())
	require.Equal(t, 0.0, lp.offeredCurrent)

	// and no switch is ever executed, no matter how long the buffer would have taken
	clock.Add(time.Minute)
	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, nil)
	require.Equal(t, 3, lp.GetPhases())
}

// TestPhaseSwitchUserOverride verifies that an explicit "Schnell" (now mode)
// cancels a pending 3p→1p switch when the physical phase already is 3p.
func TestPhaseSwitchUserOverride(t *testing.T) {
	ctrl := gomock.NewController(t)

	chargerMock := api.NewMockCharger(ctrl)
	chargerMock.EXPECT().Enable(false).Return(nil) // 0A drop from the pending sequence
	chargerMock.EXPECT().MaxCurrent(int64(18)).Return(nil)
	chargerMock.EXPECT().Enable(true).Return(nil)
	phaseMock := api.NewMockPhaseSwitcher(ctrl)
	// the switch must never happen
	phaseMock.EXPECT().Phases1p3p(gomock.Any()).Return(nil).Times(0)

	clock := clock.NewMock()
	clock.Add(time.Hour)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		wakeUpTimer: NewTimer(),
		charger: struct {
			*api.MockCharger
			*api.MockPhaseSwitcher
		}{chargerMock, phaseMock},
		minCurrent:     2,
		maxCurrent:     18,
		phases:         3,
		status:         api.StatusC,
		enabled:        true,
		offeredCurrent: 16,
	}

	// a 3p→1p sequence is pending (e.g. started by minpv)
	require.NoError(t, lp.minCharging())
	require.Equal(t, 1, lp.phaseSwitchTarget)
	require.False(t, lp.enabled)

	// user presses "Schnell": physical phase is already 3p → abort, charge at max
	require.NoError(t, lp.fastCharging())
	require.Equal(t, 0, lp.phaseSwitchTarget)
	require.Equal(t, 3, lp.phases)
	require.Equal(t, 18.0, lp.offeredCurrent)
	require.True(t, lp.enabled)
}

// TestPhaseSwitchUserOverrideViaUpdate verifies that the override is applied by
// the actual update dispatch (not only by the fastCharging function): while a
// 3p→1p sequence is pending, the user switches to "Schnell" (ModeNow). The
// dispatch must cancel the pending switch (the physical phase already matches)
// and the next cycle resumes max charging at 3p without any phase switch.
func TestPhaseSwitchUserOverrideViaUpdate(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	plainCharger := api.NewMockCharger(ctrl)
	phaseCharger := api.NewMockPhaseSwitcher(ctrl)

	chargerStatus := api.StatusC
	plainCharger.EXPECT().Status().DoAndReturn(func() (api.ChargeStatus, error) {
		return chargerStatus, nil
	}).AnyTimes()
	plainCharger.EXPECT().Enabled().Return(true, nil).AnyTimes()
	plainCharger.EXPECT().Enable(gomock.Any()).Return(nil).AnyTimes()
	plainCharger.EXPECT().MaxCurrent(gomock.Any()).Return(nil).AnyTimes()
	// the physical switch must never happen
	phaseCharger.EXPECT().Phases1p3p(gomock.Any()).Return(nil).Times(0)

	clock := clock.NewMock()

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		chargeMeter: newChargeMeter(&Null{}),
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
		phases:           3,
		phasesConfigured: 3,
		enabled:          true,
		offeredCurrent:   16,
	}

	attachListeners(t, lp)

	// a 3p→1p sequence is pending (started by pvScalePhases/minpv while
	// charging, the production path for a down-switch)
	require.NoError(t, lp.scalePhases(1))
	require.Equal(t, 1, lp.phaseSwitchTarget)
	require.False(t, lp.enabled)

	// inside the buffer: the update dispatch sees ModeNow + physical 3p →
	// the pending opposite switch is cancelled, nothing is switched
	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, nil)
	require.Equal(t, 0, lp.phaseSwitchTarget, "explicit max charging must cancel the pending opposite switch")
	require.Equal(t, 3, lp.GetPhases())

	// next cycle: max charging at 3p, no sequence, no phase switch
	lp.Update(0, 0, nil, nil, false, false, 0, nil, nil, nil)
	require.Equal(t, maxA, lp.offeredCurrent)
	require.True(t, lp.enabled)
	require.False(t, lp.phaseSwitchPending())
}

// TestPhaseSwitchHoldsOnChargingStatus verifies that a charger still reporting
// status C keeps the sequence pending even if the measured current is zero:
// the charging status is the strongest do-not-switch signal, and the driver
// additionally refuses the physical switch under load.
func TestPhaseSwitchHoldsOnChargingStatus(t *testing.T) {
	ctrl := gomock.NewController(t)

	chargerMock := api.NewMockCharger(ctrl)
	chargerMock.EXPECT().Enable(false).Return(nil)
	chargerMock.EXPECT().MaxCurrent(int64(18)).Return(nil)
	chargerMock.EXPECT().Enable(true).Return(nil)
	phaseMock := api.NewMockPhaseSwitcher(ctrl)
	phaseMock.EXPECT().Phases1p3p(3).Return(nil).Times(1)

	clock := clock.NewMock()
	clock.Add(time.Hour)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		wakeUpTimer: NewTimer(),
		charger: struct {
			*api.MockCharger
			*api.MockPhaseSwitcher
		}{chargerMock, phaseMock},
		minCurrent:     2,
		maxCurrent:     18,
		phases:         1,
		status:         api.StatusC,
		enabled:        true,
		offeredCurrent: 18,
		chargeCurrents: []float64{0, 0, 0}, // measured current is already zero
	}

	require.NoError(t, lp.fastCharging())
	require.Equal(t, 3, lp.phaseSwitchTarget)
	require.Equal(t, 0.0, lp.offeredCurrent)

	// measured current is zero but the charger still reports C: no anchor
	require.NoError(t, lp.advancePhaseSwitch())
	require.True(t, lp.phaseSwitchZeroAt.IsZero())
	clock.Add(5 * time.Minute)
	require.NoError(t, lp.advancePhaseSwitch())
	require.True(t, lp.phaseSwitchZeroAt.IsZero())
	require.Equal(t, 1, lp.phases)

	// charger goes idle: anchor, buffer, then the switch
	lp.status = api.StatusB
	require.NoError(t, lp.advancePhaseSwitch())
	require.False(t, lp.phaseSwitchZeroAt.IsZero())
	clock.Add(21 * time.Second)
	require.NoError(t, lp.advancePhaseSwitch())
	require.Equal(t, 3, lp.phases)
	require.Equal(t, 0, lp.phaseSwitchTarget)

	// next cycle: max current is restored
	require.NoError(t, lp.fastCharging())
	require.Equal(t, 18.0, lp.offeredCurrent)
	require.True(t, lp.enabled)
}

// TestPhaseSwitchAnchorResetOnCurrentReappears verifies that the 0A anchor
// restarts when current flows again during the hold: the full safety buffer
// must be spent at confirmed zero current.
func TestPhaseSwitchAnchorResetOnCurrentReappears(t *testing.T) {
	ctrl := gomock.NewController(t)

	chargerMock := api.NewMockCharger(ctrl)
	chargerMock.EXPECT().Enable(false).Return(nil)
	chargerMock.EXPECT().MaxCurrent(int64(2)).Return(nil)
	chargerMock.EXPECT().Enable(true).Return(nil)
	phaseMock := api.NewMockPhaseSwitcher(ctrl)
	phaseMock.EXPECT().Phases1p3p(1).Return(nil).Times(1)

	clock := clock.NewMock()
	clock.Add(time.Hour)

	lp := &Loadpoint{
		log:         util.NewLogger("foo"),
		bus:         evbus.New(),
		clock:       clock,
		wakeUpTimer: NewTimer(),
		charger: struct {
			*api.MockCharger
			*api.MockPhaseSwitcher
		}{chargerMock, phaseMock},
		minCurrent:     2,
		maxCurrent:     18,
		phases:         3,
		status:         api.StatusC,
		enabled:        true,
		offeredCurrent: 2,
		chargeCurrents: []float64{2, 0, 0},
	}

	// 3p→1p while charging at min current
	require.NoError(t, lp.minCharging())
	require.Equal(t, 1, lp.phaseSwitchTarget)
	require.Equal(t, 0.0, lp.offeredCurrent)

	// charger idle but current still measured (stale reading) → no anchor
	lp.status = api.StatusB
	lp.chargeCurrents = []float64{5, 0, 0}
	require.NoError(t, lp.advancePhaseSwitch())
	require.True(t, lp.phaseSwitchZeroAt.IsZero())

	// current confirmed zero → anchor is set
	lp.chargeCurrents = []float64{0, 0, 0}
	require.NoError(t, lp.advancePhaseSwitch())
	require.False(t, lp.phaseSwitchZeroAt.IsZero())

	// current reappears before the buffer elapsed → anchor restarts
	lp.chargeCurrents = []float64{5, 0, 0}
	require.NoError(t, lp.advancePhaseSwitch())
	require.True(t, lp.phaseSwitchZeroAt.IsZero())
	require.Equal(t, 3, lp.phases)

	// back to zero: the full buffer must be waited again
	lp.chargeCurrents = []float64{0, 0, 0}
	require.NoError(t, lp.advancePhaseSwitch())
	require.False(t, lp.phaseSwitchZeroAt.IsZero())
	clock.Add(10 * time.Second)
	require.NoError(t, lp.advancePhaseSwitch())
	require.Equal(t, 3, lp.phases) // inside the buffer
	clock.Add(11 * time.Second)
	require.NoError(t, lp.advancePhaseSwitch())
	require.Equal(t, 1, lp.phases)
	require.Equal(t, 0, lp.phaseSwitchTarget)

	// next cycle: min current is restored
	require.NoError(t, lp.minCharging())
	require.Equal(t, 2.0, lp.offeredCurrent)
	require.True(t, lp.enabled)
}

// TestSetPhasesHysteresisValidation verifies the setter rejects negative
// thresholds and an inverted band (including the derived defaults), and that
// a valid band is accepted and persisted.
func TestSetPhasesHysteresisValidation(t *testing.T) {
	lp := &Loadpoint{
		log:        util.NewLogger("foo"),
		bus:        evbus.New(),
		settings:   settings.NewMemorySettings(),
		minCurrent: 2,
		maxCurrent: 18,
	}

	// negative values are rejected
	require.Error(t, lp.SetPhasesHysteresis(-1, 0))
	require.Error(t, lp.SetPhasesHysteresis(0, -1))

	// down at or above the effective up threshold is rejected
	// (defaults: up = 18A*230V = 4140W, down = 2A*3*230V = 1380W)
	require.Error(t, lp.SetPhasesHysteresis(0, 5000))
	require.Error(t, lp.SetPhasesHysteresis(4000, 4000))

	// a valid band is accepted and persisted
	require.NoError(t, lp.SetPhasesHysteresis(5000, 3000))
	require.Equal(t, 5000.0, lp.PhasesUpPower)
	require.Equal(t, 3000.0, lp.PhasesDownPower)
}

// TestSetPhasesHysteresisDefaultOverlap verifies that when the derived
// defaults overlap (common for standard configurations, e.g. 6-16 A where
// 16A x 230V < 6A x 3 x 230V) the pure default (0, 0) is accepted with a
// warning - consistent with the YAML config path - while explicit inverted
// bands are still rejected.
func TestSetPhasesHysteresisDefaultOverlap(t *testing.T) {
	lp := &Loadpoint{
		log:        util.NewLogger("foo"),
		bus:        evbus.New(),
		settings:   settings.NewMemorySettings(),
		minCurrent: 6,
		maxCurrent: 16,
	}

	// defaults: up = 16A*230V = 3680W, down = 6A*3*230V = 4140W (overlapping)
	require.NoError(t, lp.SetPhasesHysteresis(0, 0))
	require.Equal(t, 0.0, lp.PhasesUpPower)
	require.Equal(t, 0.0, lp.PhasesDownPower)

	// explicit inverted bands are still rejected
	require.Error(t, lp.SetPhasesHysteresis(4000, 5000))
	require.Error(t, lp.SetPhasesHysteresis(0, 5000))
}

// TestSetPhasesMaxCurrent3Validation verifies the 3p current cap setter:
// negative values are rejected, values below minCurrent are clamped.
func TestSetPhasesMaxCurrent3Validation(t *testing.T) {
	lp := &Loadpoint{
		log:        util.NewLogger("foo"),
		bus:        evbus.New(),
		settings:   settings.NewMemorySettings(),
		minCurrent: 2,
		maxCurrent: 18,
	}

	require.Error(t, lp.SetPhasesMaxCurrent3(-1))

	// below minCurrent: clamped to minCurrent
	require.NoError(t, lp.SetPhasesMaxCurrent3(1))
	require.Equal(t, 2.0, lp.PhasesMaxCurrent3)

	// a valid cap is stored
	require.NoError(t, lp.SetPhasesMaxCurrent3(16))
	require.Equal(t, 16.0, lp.PhasesMaxCurrent3)
}

// completePendingPhaseSwitch runs a pending 0A safety sequence to completion,
// the same way the update loop would (advancePhaseSwitch): the safety buffer
// is fast-forwarded and the charger is made to report idle (status B), which
// is what the vehicle reports once the contactor opens at 0A. Used by the
// upstream tests, which exercise the scaling decisions but not the sequence
// itself (that is covered by TestPhaseSwitchSafetySequence*).
func completePendingPhaseSwitch(t *testing.T, lp *Loadpoint) {
	t.Helper()

	if !lp.phaseSwitchPending() {
		return
	}

	lp.phaseSwitchZeroAt = lp.clock.Now().Add(-phaseSwitchSafetyBuffer - time.Second)
	lp.status = api.StatusB
	require.NoError(t, lp.advancePhaseSwitch())
	require.False(t, lp.phaseSwitchPending())
}
