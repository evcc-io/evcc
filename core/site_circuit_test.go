package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

type mockDimmableMeter struct {
	api.Meter
	api.Dimmer
}

func dimmerSite(ctrl *gomock.Controller, dimmers ...api.Dimmer) *Site {
	s := &Site{log: util.NewLogger("foo")}
	for _, d := range dimmers {
		mm := &mockDimmableMeter{Meter: api.NewMockMeter(ctrl), Dimmer: d}
		s.auxMeters = append(s.auxMeters, config.NewStaticDevice[api.Meter](config.Named{}, mm))
	}
	return s
}

func TestDimming(t *testing.T) {
	for _, tc := range []struct {
		has   bool
		want  dimState
		write bool
		limit float64
	}{
		{has: false, want: dimState{}, write: false},
		{has: true, want: dimState{}, write: true},
		// active state is always written since the device may hold a different limit
		{has: false, want: dimState{active: true}, write: true, limit: 4200},
		{has: true, want: dimState{active: true}, write: true, limit: 4200},
		{has: true, want: dimState{active: true, fallback: true}, write: true, limit: 0},
	} {
		t.Logf("%+v", tc)

		ctrl := gomock.NewController(t)
		dimmer := api.NewMockDimmer(ctrl)
		s := dimmerSite(ctrl, dimmer)

		dimmer.EXPECT().Dimmed().Return(tc.has, nil)
		dimmer.EXPECT().DimPower().Return(4200.0).AnyTimes()
		if tc.write {
			dimmer.EXPECT().Dim(tc.want.active, tc.limit).Return(nil)
		}

		require.NoError(t, s.dimMeters(tc.want))

		// unchanged state is not applied again
		require.NoError(t, s.dimMeters(tc.want))
	}
}

func TestHemsDimState(t *testing.T) {
	for _, tc := range []struct {
		name   string
		limit  *float64
		powers []float64
		want   *dimState
	}{
		{"no statement", nil, nil, nil},
		{"not dimmed", new(0.0), []float64{4200}, &dimState{}},
		{"within limit", new(4200.0), []float64{4200, 0}, &dimState{active: true}},
		{"exceeds limit", new(4200.0), []float64{4200, 1000}, &dimState{active: true, fallback: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			hems := api.NewMockHEMS(ctrl)
			hems.EXPECT().MaxConsumptionPower().Return(tc.limit).AnyTimes()

			var dimmers []api.Dimmer
			for _, p := range tc.powers {
				d := api.NewMockDimmer(ctrl)
				d.EXPECT().DimPower().Return(p).AnyTimes()
				dimmers = append(dimmers, d)
			}

			s := dimmerSite(ctrl, dimmers...)
			s.hems = hems

			assert.Equal(t, tc.want, s.hemsDimState())
		})
	}
}

func TestDimStateLimit(t *testing.T) {
	ctrl := gomock.NewController(t)
	d := api.NewMockDimmer(ctrl)
	d.EXPECT().DimPower().Return(4200.0).AnyTimes()

	assert.Equal(t, 0.0, dimState{}.limit(d))
	assert.Equal(t, 4200.0, dimState{active: true}.limit(d))
	assert.Equal(t, 0.0, dimState{active: true, fallback: true}.limit(d))
}
