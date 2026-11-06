package core

// Fork extension (twc3shelly): 0A safety sequence for phase switching.
//
// Switching phases under load can damage the installation contactor (Shelly
// Pro 1PM Gen3) and the vehicle's charging electronics, so the load is
// dropped to 0A first, held for the safety buffer (phaseSwitchSafetyBuffer =
// 20s) and only then is the physical switch executed. After the switch the
// pv logic re-establishes the charge current from the current surplus.
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

	// buffer elapsed but the charger still reports charging → no switch under load
	clock.Add(11 * time.Second)
	require.NoError(t, lp.advancePhaseSwitch())
	require.Equal(t, 1, lp.phases)

	// charger idle now (status B: vehicle connected, contactor open) → switch
	lp.status = api.StatusB
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

	// buffer elapsed, charger idle → switch
	clock.Add(21 * time.Second)
	lp.status = api.StatusB
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
