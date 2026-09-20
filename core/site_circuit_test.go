package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

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
		want *float64
	}{
		// nil: circuit has no opinion - the device must not be touched, so a
		// limit configured outside evcc is preserved (issue #30068)
		{has: false, want: nil},
		{has: true, want: nil},
		{has: false, want: new(0.0)},
		{has: false, want: new(4200.0)},
		{has: true, want: new(0.0)},
		// active limit is re-stated: its value may have changed
		{has: true, want: new(4200.0)},
	} {
		t.Logf("%+v", tc)

		if tc.want != nil {
			dimmer.EXPECT().Dimmed().Return(tc.has, nil)
			if tc.has || *tc.want > 0 {
				dimmer.EXPECT().Dim(*tc.want).Return(nil)
			}

			s.dimLimit = nil
			require.NoError(t, s.dimMeters(*tc.want))
		}

		if !ctrl.Satisfied() {
			ctrl.Finish()
		}
	}
}
