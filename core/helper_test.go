package core

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestCircuitMaxPower(t *testing.T) {
	ctrl := gomock.NewController(t)

	chain := func(limits ...float64) api.Circuit {
		var parent api.Circuit
		for i := len(limits) - 1; i >= 0; i-- {
			c := api.NewMockCircuit(ctrl)
			c.EXPECT().EffectiveMaxPower().Return(limits[i]).AnyTimes()
			c.EXPECT().GetParent().Return(parent).AnyTimes()
			parent = c
		}
		return parent
	}

	for _, tc := range []struct {
		name   string
		limits []float64 // child first
		res    float64
	}{
		{"no circuit", nil, 0},
		{"unlimited", []float64{0, 0}, 0},
		{"own limit", []float64{11000, 0}, 11000},
		{"lower parent limit", []float64{11000, 7000}, 7000},
		{"unlimited child", []float64{0, 7000}, 7000},
		{"lower child limit", []float64{4000, 7000}, 4000},
	} {
		assert.Equal(t, tc.res, circuitMaxPower(chain(tc.limits...)), tc.name)
	}
}
