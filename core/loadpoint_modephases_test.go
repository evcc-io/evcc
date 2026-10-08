package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// newModePresetLoadpoint builds a loadpoint with the given charger for mode preset tests.
func newModePresetLoadpoint(t *testing.T, store settings.Settings, charger api.Charger) *Loadpoint {
	t.Helper()

	uiChan, pushChan, lpChan := createChannels(t)

	lp := NewLoadpoint(util.NewLogger("foo"), store)
	lp.charger = charger
	lp.Prepare(new(Site), uiChan, pushChan, lpChan)

	return lp
}

func modePresetCharger(ctrl *gomock.Controller) api.Charger {
	plainCharger := api.NewMockCharger(ctrl)
	plainCharger.EXPECT().Enabled().Return(false, nil).AnyTimes()
	return struct {
		*api.MockCharger
		*api.MockPhaseSwitcher
	}{plainCharger, api.NewMockPhaseSwitcher(ctrl)}
}

// TestModePhasePreset asserts that switching charge mode applies the
// configured per-mode phase preset without touching the charger directly.
// The actual hardware switch is left to the control loop (scalePhasesRequired).
func TestModePhasePreset(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := settings.NewMemorySettings()
	lp := newModePresetLoadpoint(t, store, modePresetCharger(ctrl))

	require.NoError(t, lp.SetPhasesSmart(1))
	require.NoError(t, lp.SetPhasesNow(3))

	// smart applies 1p preset
	lp.SetMode(api.ModeSmart)
	require.Equal(t, 1, lp.GetPhasesConfigured())

	// now applies 3p preset
	lp.SetMode(api.ModeNow)
	require.Equal(t, 3, lp.GetPhasesConfigured())

	// cleared preset leaves configured phases untouched
	require.NoError(t, lp.SetPhasesSmart(0))
	lp.SetMode(api.ModeSmart)
	require.Equal(t, 3, lp.GetPhasesConfigured())

	// off has no preset
	lp.SetMode(api.ModeOff)
	require.Equal(t, 3, lp.GetPhasesConfigured())

	ctrl.Finish()
}

// TestModePhasePresetValidation asserts preset value validation.
func TestModePhasePresetValidation(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := settings.NewMemorySettings()
	lp := newModePresetLoadpoint(t, store, modePresetCharger(ctrl))

	require.Error(t, lp.SetPhasesSmart(2))
	require.Error(t, lp.SetPhasesNow(2))
	require.NoError(t, lp.SetPhasesSmart(0))
	require.Equal(t, 0, lp.GetPhasesSmart())

	ctrl.Finish()
}

// TestModePhasePresetIgnoredWithoutSwitching asserts that presets are stored
// but never applied on chargers without phase switching support.
func TestModePhasePresetIgnoredWithoutSwitching(t *testing.T) {
	ctrl := gomock.NewController(t)
	plainCharger := api.NewMockCharger(ctrl)
	plainCharger.EXPECT().Enabled().Return(false, nil).AnyTimes()

	store := settings.NewMemorySettings()
	lp := newModePresetLoadpoint(t, store, plainCharger)

	require.NoError(t, lp.SetPhasesSmart(1))
	require.NoError(t, lp.SetPhasesNow(3))

	lp.SetMode(api.ModeSmart)
	require.Equal(t, 0, lp.GetPhasesConfigured())

	lp.SetMode(api.ModeNow)
	require.Equal(t, 0, lp.GetPhasesConfigured())

	ctrl.Finish()
}

// TestModePhasePresetRestore asserts that presets survive a restart via settings.
func TestModePhasePresetRestore(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := settings.NewMemorySettings()
	lp := newModePresetLoadpoint(t, store, modePresetCharger(ctrl))

	require.NoError(t, lp.SetPhasesSmart(1))
	require.NoError(t, lp.SetPhasesNow(3))

	restored := newModePresetLoadpoint(t, store, modePresetCharger(ctrl))
	restored.restoreSettings()

	require.Equal(t, 1, restored.GetPhasesSmart())
	require.Equal(t, 3, restored.GetPhasesNow())

	ctrl.Finish()
}
