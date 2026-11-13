package core

import (
	"testing"

	"github.com/benbjohnson/clock"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/grid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// phaseMeter returns a charge meter that measures phase currents
func phaseMeter(t *testing.T) *chargeMeter {
	ctrl := gomock.NewController(t)
	return newChargeMeter(struct {
		*api.MockMeter
		*api.MockPhaseCurrents
	}{api.NewMockMeter(ctrl), api.NewMockPhaseCurrents(ctrl)})
}

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
		lp := &Loadpoint{log: util.NewLogger("foo"), chargeMeter: phaseMeter(t), phaseRotation: tc.rotation}
		assert.Equal(t, tc.res, lp.gridPhases(tc.phases), tc)
	}
}

// the mapping is only used with measured phase currents and a grid with neutral
func TestLoadpointGridPhasesIgnored(t *testing.T) {
	l3 := api.GridPhases{false, false, true}

	lp := &Loadpoint{log: util.NewLogger("foo"), chargeMeter: newChargeMeter(&Null{}), phaseRotation: 3}
	assert.Equal(t, api.AllGridPhases, lp.gridPhases(1), "charger without phase currents")

	lp = &Loadpoint{log: util.NewLogger("foo"), chargeMeter: phaseMeter(t), phaseRotation: 3}
	assert.Equal(t, l3, lp.gridPhases(1), "measured")

	grid.SetNoNeutral(true)
	defer grid.SetNoNeutral(false)
	assert.Equal(t, api.AllGridPhases, lp.gridPhases(1), "IT grid")
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
				log:            util.NewLogger("foo"),
				clock:          clock.NewMock(),
				chargeMeter:    phaseMeter(t),
				phaseRotation:  tc.rotation,
				chargeCurrents: tc.currents,
				offeredCurrent: 6,
			}
			assert.Equal(t, tc.res, lp.GetGridPhaseCurrents())
		})
	}
}

func TestLoadpointCircuitCurrents(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rotation int
		measured bool
		currents []float64
		res      [3]float64
		ok       bool
	}{
		{"unmapped", 0, true, []float64{16, 0, 0}, [3]float64{16, 16, 16}, false},
		{"mapped to L3", 3, true, []float64{16, 0, 0}, [3]float64{0, 0, 16}, true},
		{"mapped asymmetric", 1, true, []float64{16, 6, 6}, [3]float64{16, 6, 6}, true},
		{"mapped without phase currents", 3, false, nil, [3]float64{6, 6, 6}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			meter := newChargeMeter(&Null{})
			if tc.measured {
				meter = phaseMeter(t)
			}

			lp := &Loadpoint{
				log:            util.NewLogger("foo"),
				clock:          clock.NewMock(),
				chargeMeter:    meter,
				phaseRotation:  tc.rotation,
				chargeCurrents: tc.currents,
				offeredCurrent: 6,
				status:         api.StatusC,
			}

			assert.Equal(t, tc.res, lp.circuitCurrents())

			currents, ok := lp.MeasuredGridPhaseCurrents()
			assert.Equal(t, tc.ok, ok)
			if ok {
				assert.Equal(t, tc.res, currents)
			}
		})
	}
}

// otherLoad is a load on the circuit that is not a loadpoint
type otherLoad struct {
	circuit  api.Circuit
	currents [3]float64
}

func (l *otherLoad) GetChargePower() float64 { return 0 }
func (l *otherLoad) GetMaxPhaseCurrent() float64 {
	return max(l.currents[0], l.currents[1], l.currents[2])
}
func (l *otherLoad) GetGridPhaseCurrents() [3]float64 { return l.currents }
func (l *otherLoad) GetCircuit() api.Circuit          { return l.circuit }
func (l *otherLoad) MeasuredGridPhaseCurrents() ([3]float64, bool) {
	return l.currents, true
}

// switching a mapped 1p loadpoint to 3p must respect the load already on the new phases
func TestLoadpointCircuitAllowsPhasesMapped(t *testing.T) {
	for _, tc := range []struct {
		name  string
		other float64 // current of another load on L3
		res   bool
	}{
		{"room on L3", 14, true},
		{"L3 too full for min current", 15, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			c, err := circuit.New(util.NewLogger("foo"), "foo", 20, 0, nil, 0)
			require.NoError(t, err)

			lp := &Loadpoint{
				log:            util.NewLogger("foo"),
				clock:          clock.NewMock(),
				circuit:        c,
				chargeMeter:    phaseMeter(t),
				phaseRotation:  1,
				chargeCurrents: []float64{16, 0, 0},
				offeredCurrent: 16,
				status:         api.StatusC,
				enabled:        true,
			}

			require.NoError(t, c.Update([]api.CircuitLoad{lp, &otherLoad{circuit: c, currents: [3]float64{0, 0, tc.other}}}))
			assert.Equal(t, tc.res, lp.circuitAllowsPhases(3, 6))
		})
	}
}
