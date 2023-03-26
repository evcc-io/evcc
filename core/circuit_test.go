package core

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

type testMeter struct {
	cur float64
}

// interface Meter
func (dm *testMeter) CurrentPower() (float64, error) {
	return dm.cur * (11 / 16), nil
}

// interface MeterCurrents
func (dm *testMeter) Currents() (float64, float64, float64, error) {
	return dm.cur, dm.cur, dm.cur, nil
}

func TestCurrentCircuitMeter(t *testing.T) {
	limit := 20.0
	mtr := &testMeter{cur: 0.0}
	circ := NewCircuit(util.NewLogger("foo"), limit, nil, mtr)
	assert.NotNilf(t, circ, "circuit not created")

	// no consumption
	curAv, err := circ.MaxPhasesCurrent()
	assert.Equal(t, curAv, 0.0)
	assert.Nil(t, err)

	// no consumption from meter, return limit
	curAv = circ.GetRemainingCurrent()
	assert.Equal(t, limit, curAv)

	// set some consumption on meter
	mtr.cur = 5
	curAv = circ.GetRemainingCurrent()
	assert.Equal(t, limit-mtr.cur, curAv)

	// simulate production in circuit (negative consumption)
	// available current is limit - consumption
	mtr.cur = -5
	curAv = circ.GetRemainingCurrent()
	assert.Equal(t, limit-mtr.cur, curAv)
}

func TestParentCircuitHierarchy(t *testing.T) {
	// two circuits, check limit and consumption from both sides
	limitMain := 20.0
	circMain := NewCircuit(util.NewLogger("main"), limitMain, nil, &testMeter{cur: 16.0})
	assert.NotNilf(t, circMain, "circuit not created")

	// add subcircuit with meter
	limitSub := 20.0
	circSub := NewCircuit(util.NewLogger("sub"), limitSub, circMain, &testMeter{cur: 10.0})

	assert.NotNilf(t, circSub.parentCircuit, "parent circuit not set")
	assert.NotNilf(t, circSub.phaseCurrents, "sub circuit meter not set")

	curAv, err := circSub.MaxPhasesCurrent()
	assert.Equal(t, curAv, 10.0)
	assert.Nil(t, err)

	// remaining from sub circuit shall return the lower remaining from main
	// sub uses 10 out of limit 20 - remain = 10
	// main uses 16 out of limit 20 - remain = 4
	assert.Equal(t, circMain.GetRemainingCurrent(), 4.0)
	assert.Equal(t, circSub.GetRemainingCurrent(), 4.0)

	// increasing the limit of main. Main has more remaining, sub limit is relevant
	circMain.maxCurrent = 30
	assert.Equal(t, circMain.GetRemainingCurrent(), 14.0)
	assert.Equal(t, circSub.GetRemainingCurrent(), 10.0)
}
