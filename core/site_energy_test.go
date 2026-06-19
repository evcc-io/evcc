package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

// TestCollectMetersZeroEnergy verifies that a spurious zero energy reading is
// dropped (kept nil) so the persisted history retains the last value (#30950).
func TestCollectMetersZeroEnergy(t *testing.T) {
	for _, tc := range []struct {
		name string
		read float64
		want *float64
	}{
		{"zero is skipped", 0, nil},
		{"non-zero passes through", 1234.5, new(1234.5)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)

			meter := api.NewMockMeter(ctrl)
			meter.EXPECT().CurrentPower().Return(100.0, nil)

			energy := api.NewMockMeterEnergy(ctrl)
			energy.EXPECT().TotalEnergy().Return(tc.read, nil)

			mm := &struct {
				api.Meter
				api.MeterEnergy
			}{
				Meter:       meter,
				MeterEnergy: energy,
			}

			site := &Site{log: util.NewLogger("foo")}
			res := site.collectMeters("pv", []config.Device[api.Meter]{
				config.NewStaticDevice[api.Meter](config.Named{}, mm),
			})

			assert.Equal(t, tc.want, res[0].Energy)
			assert.Equal(t, 100.0, res[0].Power)
		})
	}
}
