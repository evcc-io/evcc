package core

import (
	"testing"

	evbus "github.com/asaskevich/EventBus"
	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestSyncCharger(t *testing.T) {
	tc := []struct {
		status                      api.ChargeStatus
		expected, actual, corrected bool
	}{
		{api.StatusA, false, false, false},
		{api.StatusC, false, false, true}, // disabled but charging
		{api.StatusA, false, true, true},
		{api.StatusA, true, false, false},
		{api.StatusA, true, true, true},
	}

	ctrl := gomock.NewController(t)

	for _, tc := range tc {
		t.Logf("%+v", tc)

		charger := api.NewMockCharger(ctrl)
		charger.EXPECT().Enabled().Return(tc.actual, nil).AnyTimes()

		if tc.status == api.StatusC {
			charger.EXPECT().Enable(tc.corrected).Times(1)
		}

		lp := &Loadpoint{
			log:     util.NewLogger("foo"),
			clock:   clock.New(),
			charger: charger,
			status:  tc.status,
			enabled: tc.expected,
		}

		require.NoError(t, lp.syncCharger())
		assert.Equal(t, tc.corrected, lp.enabled)
	}
}

func TestSyncChargerCurrentsByGetter(t *testing.T) {
	tc := []struct {
		lpCurrent, actualCurrent, outCurrent float64
	}{
		{6, 5, 5}, // force
		{6, 6.1, 6},
		{6, 6.5, 6.5},
		{6, 7, 7},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)
		t.Logf("%+v", tc)

		ch := api.NewMockCharger(ctrl)
		cg := api.NewMockCurrentGetter(ctrl)

		charger := struct {
			api.Charger
			api.CurrentGetter
		}{
			ch, cg,
		}

		ch.EXPECT().Enabled().Return(true, nil)
		cg.EXPECT().GetMaxCurrent().Return(tc.actualCurrent, nil).MaxTimes(1)

		lp := &Loadpoint{
			log:           util.NewLogger("foo"),
			bus:           evbus.New(),
			clock:         clock.New(),
			charger:       charger,
			status:        api.StatusC,
			enabled:       true,
			phases:        3,
			chargeCurrent: tc.lpCurrent,
		}

		require.NoError(t, lp.syncCharger())
		assert.Equal(t, tc.outCurrent, lp.chargeCurrent)
	}
}

func TestSyncChargerCurrentsByMeasurement(t *testing.T) {
	tc := []struct {
		lpCurrent     float64
		actualCurrent float64
		outCurrent    float64
	}{
		{6, 5, 6}, // ignore
		{6, 6.1, 6},
		{6, 6.5, 6},
		{6, 7, 6}, // ignore
		{6, 7.1, 7.1},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)
		t.Logf("%+v", tc)

		charger := api.NewMockCharger(ctrl)
		charger.EXPECT().Enabled().Return(true, nil)

		lp := &Loadpoint{
			log:            util.NewLogger("foo"),
			bus:            evbus.New(),
			clock:          clock.New(),
			charger:        charger,
			status:         api.StatusC,
			enabled:        true,
			phases:         3,
			chargeCurrent:  tc.lpCurrent,
			chargeCurrents: []float64{tc.actualCurrent, 0, 0},
		}

		require.NoError(t, lp.syncCharger())
		assert.Equal(t, tc.outCurrent, lp.chargeCurrent)
	}
}

func TestSyncChargerPhasesByGetter(t *testing.T) {
	tc := []struct {
		lpPhases, actualPhases, outPhases int
	}{
		{0, 0, 0},
		{1, 0, 1},
		{1, 1, 1},
		{1, 3, 3},
		{3, 0, 3},
		{3, 1, 1}, // force
		{3, 3, 3},
	}

	for _, tc := range tc {
		ctrl := gomock.NewController(t)
		t.Logf("%+v", tc)

		ch := api.NewMockCharger(ctrl)
		ps := api.NewMockPhaseSwitcher(ctrl)
		pg := api.NewMockPhaseGetter(ctrl)

		charger := struct {
			api.Charger
			api.PhaseSwitcher
			api.PhaseGetter
		}{
			ch, ps, pg,
		}

		ch.EXPECT().Enabled().Return(true, nil)
		pg.EXPECT().GetPhases().Return(tc.actualPhases, nil).MaxTimes(1)

		lp := &Loadpoint{
			log:            util.NewLogger("foo"),
			bus:            evbus.New(),
			clock:          clock.New(),
			charger:        charger,
			status:         api.StatusC,
			enabled:        true,
			phases:         tc.lpPhases,
			measuredPhases: tc.actualPhases,
		}

		require.NoError(t, lp.syncCharger())
		assert.Equal(t, tc.outPhases, lp.phases)
	}
}

func TestSyncChargerPhasesByMeasurement(t *testing.T) {
	tc := []struct {
		lpPhases, actualPhases, outPhases int
	}{
		{0, 0, 0},
		{1, 0, 1},
		{1, 1, 1},
		{1, 3, 3},
		{3, 0, 3},
		{3, 1, 3}, // ignore
		{3, 3, 3},
	}

	ctrl := gomock.NewController(t)

	for _, tc := range tc {
		t.Logf("%+v", tc)

		ch := api.NewMockCharger(ctrl)
		ps := api.NewMockPhaseSwitcher(ctrl)

		charger := struct {
			api.Charger
			api.PhaseSwitcher
		}{
			ch, ps,
		}

		ch.EXPECT().Enabled().Return(true, nil)

		lp := &Loadpoint{
			log:            util.NewLogger("foo"),
			bus:            evbus.New(),
			clock:          clock.New(),
			charger:        charger,
			status:         api.StatusC,
			enabled:        true,
			phases:         tc.lpPhases,
			measuredPhases: tc.actualPhases,
		}

		require.NoError(t, lp.syncCharger())
		assert.Equal(t, tc.outPhases, lp.phases)
	}
}
