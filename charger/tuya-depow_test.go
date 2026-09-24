package charger

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
)

func TestTuyaDepowStep(t *testing.T) {
	for _, tc := range []struct {
		current, expected int64
	}{
		{5, 6}, {6, 6}, {7, 6}, {8, 8}, {9, 8}, {12, 10}, {13, 13}, {15, 13}, {16, 16}, {32, 16},
	} {
		assert.Equal(t, tc.expected, tuyaDepowStep(tuyaDepowDefaultSteps, tc.current), tc.current)
	}
}

func TestTuyaDepowStatus(t *testing.T) {
	for _, tc := range []struct {
		code     any
		expected api.ChargeStatus
	}{
		{float64(100), api.StatusA}, {float64(101), api.StatusA},
		{float64(200), api.StatusB}, {float64(204), api.StatusB},
		{float64(300), api.StatusC},
		{float64(501), api.StatusNone}, {nil, api.StatusNone},
	} {
		res, err := tuyaDepowStatus(tc.code)
		assert.Equal(t, tc.expected, res, tc.code)
		assert.Equal(t, tc.expected == api.StatusNone, err != nil, tc.code)
	}
}
