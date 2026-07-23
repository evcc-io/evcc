package core

import (
	"errors"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// curtailSiteWithMeter builds a Site with a single curtailable PV meter.
func curtailSiteWithMeter(m *curtailableMeter) *Site {
	return &Site{
		log:      util.NewLogger("foo"),
		pvMeters: []config.Device[api.Meter]{config.NewStaticDevice[api.Meter](config.Named{}, api.Meter(m))},
	}
}

func TestCurtailPV(t *testing.T) {
	m := &curtailableMeter{percent: 100}
	s := curtailSiteWithMeter(m)

	// first apply writes
	require.NoError(t, s.curtailPV(new(60)))
	require.Equal(t, []int{60}, m.setCalls)

	// unchanged: no write
	require.NoError(t, s.curtailPV(new(60)))
	require.Equal(t, []int{60}, m.setCalls)

	// changed: writes
	require.NoError(t, s.curtailPV(new(100)))
	require.Equal(t, []int{60, 100}, m.setCalls)
}

func TestCurtailPVReapplyOnError(t *testing.T) {
	m := &curtailableMeter{percent: 100}
	s := curtailSiteWithMeter(m)

	// error: cache not advanced
	m.setErr = errors.New("nope")
	require.Error(t, s.curtailPV(new(0)))

	// same percent reapplied because previous attempt failed
	m.setErr = nil
	require.NoError(t, s.curtailPV(new(0)))
	require.Equal(t, []int{0, 0}, m.setCalls)
}

func TestRevertSmartFeedInCurtail(t *testing.T) {
	// inactive: nothing happens
	m := &curtailableMeter{percent: 100}
	s := curtailSiteWithMeter(m)
	require.NoError(t, s.revertSmartFeedInCurtail())
	require.Empty(t, m.setCalls)

	// active without HEMS: restore to uncurtailed (100%)
	m = &curtailableMeter{percent: 0}
	s = curtailSiteWithMeter(m)
	s.smartFeedInDisableActive = true
	s.curtailPercent = new(0) // feed-in previously curtailed to 0%
	require.NoError(t, s.revertSmartFeedInCurtail())
	require.Equal(t, []int{100}, m.setCalls)
}

func TestDimming(t *testing.T) {
	ctrl := gomock.NewController(t)

	meter := api.NewMockMeter(ctrl)
	dimmer := api.NewMockDimmer(ctrl)
	mm := &struct {
		api.Meter
		api.Dimmer
	}{
		Meter:  meter,
		Dimmer: dimmer,
	}

	s := &Site{
		log:       util.NewLogger("foo"),
		auxMeters: []config.Device[api.Meter]{config.NewStaticDevice[api.Meter](config.Named{}, mm)},
	}

	for _, tc := range []struct {
		has  bool
		want *bool
	}{
		// nil: circuit has no opinion - the device must not be touched, so a
		// limit configured outside evcc is preserved (issue #30068)
		{has: false, want: nil},
		{has: true, want: nil},
		{has: false, want: new(false)},
		{has: false, want: new(true)},
		{has: true, want: new(false)},
		{has: true, want: new(true)},
	} {
		t.Logf("%+v", tc)

		if tc.want != nil {
			dimmer.EXPECT().Dimmed().Return(tc.has, nil)
			if tc.has != *tc.want {
				dimmer.EXPECT().Dim(*tc.want).Return(nil)
			}

			require.NoError(t, s.dimMeters(*tc.want))
		}

		if !ctrl.Satisfied() {
			ctrl.Finish()
		}
	}
}
