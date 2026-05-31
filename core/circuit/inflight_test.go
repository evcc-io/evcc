package circuit

import (
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// circuitLoadStub is a minimal api.CircuitLoad for in-flight tests.
type circuitLoadStub struct {
	circuit                                        api.Circuit
	power, current, inflightPower, inflightCurrent float64
}

func (s *circuitLoadStub) GetChargePower() float64     { return s.power }
func (s *circuitLoadStub) GetMaxPhaseCurrent() float64 { return s.current }
func (s *circuitLoadStub) GetInflightPower() float64   { return s.inflightPower }
func (s *circuitLoadStub) GetInflightCurrent() float64 { return s.inflightCurrent }
func (s *circuitLoadStub) GetCircuit() api.Circuit     { return s.circuit }

// TestCircuitInflightCurrent verifies that a loadpoint's in-flight (un-metered)
// current reduces the headroom another actuation may take, preventing a
// transient overshoot of the circuit limit.
func TestCircuitInflightCurrent(t *testing.T) {
	c, err := New(util.NewLogger("foo"), "foo", 10, 0, nil, 0) // maxCurrent 10, no meter
	require.NoError(t, err)

	lp := &circuitLoadStub{circuit: c, current: 4, inflightCurrent: 3}
	require.NoError(t, c.Update([]api.CircuitLoad{lp}))

	// metered 4 + in-flight 3 = 7 used -> headroom 3
	assert.Equal(t, 3.0, c.ValidateCurrent(0, 5), "should cap to remaining headroom")
	assert.Equal(t, 2.0, c.ValidateCurrent(0, 2), "within headroom -> unchanged")

	// once the reserve clears, the full metered headroom (10-4=6) returns
	lp.inflightCurrent = 0
	require.NoError(t, c.Update([]api.CircuitLoad{lp}))
	assert.Equal(t, 5.0, c.ValidateCurrent(0, 5), "no reserve -> full headroom")
}

// TestCircuitInflightPower mirrors the current test for power-limited circuits.
func TestCircuitInflightPower(t *testing.T) {
	c, err := New(util.NewLogger("foo"), "foo", 0, 10000, nil, 0) // maxPower 10kW, no meter
	require.NoError(t, err)

	lp := &circuitLoadStub{circuit: c, power: 4000, inflightPower: 3000}
	require.NoError(t, c.Update([]api.CircuitLoad{lp}))

	assert.Equal(t, 3000.0, c.ValidatePower(0, 5000), "should cap to remaining headroom")

	lp.inflightPower = 0
	require.NoError(t, c.Update([]api.CircuitLoad{lp}))
	assert.Equal(t, 5000.0, c.ValidatePower(0, 5000), "no reserve -> full headroom")
}

// TestCircuitInflightHierarchy verifies in-flight reserves propagate to parents.
func TestCircuitInflightHierarchy(t *testing.T) {
	log := util.NewLogger("foo")
	parent, err := New(log, "parent", 10, 0, nil, 0)
	require.NoError(t, err)
	child, err := New(log, "child", 10, 0, nil, 0)
	require.NoError(t, err)
	require.NoError(t, child.Wrap(parent))

	lp := &circuitLoadStub{circuit: child, current: 4, inflightCurrent: 3}
	require.NoError(t, parent.Update([]api.CircuitLoad{lp}))

	// parent must see the child's in-flight reserve too
	assert.Equal(t, 3.0, parent.GetInflightCurrent())
	assert.Equal(t, 3.0, parent.ValidateCurrent(0, 5), "parent caps to child headroom")
}

// TestCircuitInflightMetered verifies that for a circuit with its own (lagging)
// meter, the loadpoints' in-flight reserves are still added on top of the
// metered value.
func TestCircuitInflightMetered(t *testing.T) {
	ctrl := gomock.NewController(t)
	meter := api.NewMockMeter(ctrl)

	c, err := New(util.NewLogger("foo"), "foo", 0, 10000, meter, 0) // maxPower 10kW, metered
	require.NoError(t, err)

	lp := &circuitLoadStub{circuit: c, inflightPower: 3000}

	// metered 4kW + in-flight 3kW = 7kW used -> headroom 3kW
	meter.EXPECT().CurrentPower().Return(4000.0, nil)
	require.NoError(t, c.Update([]api.CircuitLoad{lp}))

	assert.Equal(t, 3000.0, c.GetInflightPower(), "in-flight aggregated on a metered circuit")
	assert.Equal(t, 3000.0, c.ValidatePower(0, 5000), "metered + in-flight headroom")
}
