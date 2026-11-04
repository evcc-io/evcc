package core

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePhaseRotation(t *testing.T) {
	for in, res := range map[string]int{"": 0, "L1": 1, "l2": 2, " L3 ": 3} {
		r, err := parsePhaseRotation(in)
		require.NoError(t, err, in)
		assert.Equal(t, res, r, in)
	}

	for _, in := range []string{"L4", "1", "RST"} {
		_, err := parsePhaseRotation(in)
		assert.Error(t, err, in)
	}
}

func TestLoadpointGridPhases(t *testing.T) {
	for _, tc := range []struct {
		rotation, phases int
		res              api.GridPhases
	}{
		{0, 1, api.AllGridPhases},
		{1, 1, api.GridPhases{true, false, false}},
		{2, 1, api.GridPhases{false, true, false}},
		{3, 1, api.GridPhases{false, false, true}},
		{3, 2, api.GridPhases{true, false, true}},
		{3, 3, api.AllGridPhases},
		{3, 0, api.AllGridPhases},
	} {
		lp := &Loadpoint{phaseRotation: tc.rotation}
		assert.Equal(t, tc.res, lp.gridPhases(tc.phases), tc)
	}
}

func TestLoadpointGridPhaseCurrents(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rotation int
		currents []float64
		res      [3]float64
	}{
		{"unknown rotation", 0, []float64{14.9, 0, 0}, [3]float64{14.9, 14.9, 14.9}},
		{"no rotation", 1, []float64{14.9, 0, 0}, [3]float64{14.9, 0, 0}},
		{"rotated to L3", 3, []float64{14.9, 0, 0}, [3]float64{0, 0, 14.9}},
		{"rotated 3p", 2, []float64{1, 2, 3}, [3]float64{3, 1, 2}},
		{"not measured", 3, nil, [3]float64{6, 6, 6}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			lp := &Loadpoint{
				clock:          clock.NewMock(),
				phaseRotation:  tc.rotation,
				chargeCurrents: tc.currents,
				offeredCurrent: 6,
			}
			assert.Equal(t, tc.res, lp.GetGridPhaseCurrents())
		})
	}
}
