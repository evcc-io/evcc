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
		has, want bool
	}{
		{has: false, want: false},
		{has: false, want: true},
		{has: true, want: false},
		{has: true, want: true},
	} {
		t.Logf("%+v", tc)

		dimmer.EXPECT().Dimmed().Return(tc.has, nil)
		if tc.has != tc.want {
			dimmer.EXPECT().Dim(tc.want).Return(nil)
		}

		require.NoError(t, s.dimMeters(tc.want))

		if !ctrl.Satisfied() {
			ctrl.Finish()
		}
	}
}
