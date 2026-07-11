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

// curtailSite builds a Site with a single PV meter that also implements api.Curtailer
func curtailSite(t *testing.T) (*Site, *api.MockCurtailer) {
	t.Helper()
	ctrl := gomock.NewController(t)

	cm := &struct {
		api.Meter
		api.Curtailer
	}{
		Meter:     api.NewMockMeter(ctrl),
		Curtailer: api.NewMockCurtailer(ctrl),
	}

	s := &Site{
		log:      util.NewLogger("foo"),
		pvMeters: []config.Device[api.Meter]{config.NewStaticDevice[api.Meter](config.Named{}, cm)},
	}

	return s, cm.Curtailer.(*api.MockCurtailer)
}

func TestCurtailPV(t *testing.T) {
	s, mc := curtailSite(t)

	// first apply writes
	mc.EXPECT().SetCurtailPercent(60).Return(nil)
	require.NoError(t, s.curtailPV(new(60)))

	// unchanged: no write
	require.NoError(t, s.curtailPV(new(60)))

	// changed: writes
	mc.EXPECT().SetCurtailPercent(100).Return(nil)
	require.NoError(t, s.curtailPV(new(100)))
}

func TestCurtailPVReapplyOnError(t *testing.T) {
	s, mc := curtailSite(t)

	// error: cache not advanced
	mc.EXPECT().SetCurtailPercent(0).Return(errors.New("nope"))
	require.Error(t, s.curtailPV(new(0)))

	// same percent reapplied because previous attempt failed
	mc.EXPECT().SetCurtailPercent(0).Return(nil)
	require.NoError(t, s.curtailPV(new(0)))
}

func TestRevertSmartFeedInCurtail(t *testing.T) {
	// inactive: nothing happens
	s, _ := curtailSite(t)
	require.NoError(t, s.revertSmartFeedInCurtail())

	// active without HEMS: restore to uncurtailed (100%)
	s, mc := curtailSite(t)
	s.smartFeedInDisableActive = true
	s.curtailPercent = new(0) // feed-in previously curtailed to 0%
	mc.EXPECT().SetCurtailPercent(100).Return(nil)
	require.NoError(t, s.revertSmartFeedInCurtail())
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
		}

		require.NoError(t, s.dimMeters(tc.want))

		if !ctrl.Satisfied() {
			ctrl.Finish()
		}
	}
}
