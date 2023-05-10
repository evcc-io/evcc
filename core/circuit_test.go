package core

import (
	"math"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

// testMeter is for circuit testing.
type testMeter struct {
	cur float64
	pwr float64
}

// interface Meter
func (dm *testMeter) CurrentPower() (float64, error) {
	return dm.pwr, nil
}

// interface MeterCurrents
func (dm *testMeter) Currents() (float64, float64, float64, error) {
	return dm.cur, dm.cur, dm.cur, nil
}

func TestCircuitMeterCurrent(t *testing.T) {
	limit := 20.0
	mtr := &testMeter{cur: 0.0}
	circ := NewCircuit(util.NewLogger("foo"), limit, 0, nil, mtr, mtr)
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

func TestCircuitMeter_noCheck(t *testing.T) {
	Voltage = 236
	mtr := &testMeter{cur: 5.0, pwr: 1000}
	circ := NewCircuit(util.NewLogger("foo"), 0, 0, nil, mtr, mtr)
	assert.NotNilf(t, circ, "circuit not created")

	// initilized with 0 should give maxFloat as remaining current / power
	curAv, err := circ.MaxPhasesCurrent()
	assert.Nil(t, err)

	assert.Equal(t, curAv, mtr.cur)
	curRem := circ.GetRemainingCurrent()
	assert.Equal(t, curRem, math.MaxFloat64)

	// power remaining should return maxFloat for remaining power
	curPwr, err := circ.CurrentPower()
	assert.Nil(t, err)
	assert.Equal(t, curPwr, mtr.pwr)
	pwrRem := circ.GetRemainingPower()
	assert.Equal(t, pwrRem, math.MaxFloat64)
}

func TestHierarchyCurrent(t *testing.T) {
	// two circuits, check limit and consumption from both sides
	limitMain := 20.0
	tstMtrMain := testMeter{cur: 16.0}
	circMain := NewCircuit(util.NewLogger("main"), limitMain, 0, nil, &tstMtrMain, &tstMtrMain)
	assert.NotNilf(t, circMain, "circuit not created")

	// add subcircuit with meter
	limitSub := 20.0
	tstMtrSub := testMeter{cur: 10.0}
	circSub := NewCircuit(util.NewLogger("sub"), limitSub, 0, circMain, &tstMtrSub, &tstMtrSub)

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

func TestCircuitMeterPower(t *testing.T) {
	limit := 20000.0
	mtr := &testMeter{cur: 0, pwr: 0} // power will be calculated from current and phases
	circ := NewCircuit(util.NewLogger("foo"), 0, limit, nil, mtr, mtr)
	assert.NotNilf(t, circ, "circuit not created")

	// no consumption
	pwrAv, err := circ.CurrentPower()
	assert.Equal(t, pwrAv, 0.0)
	assert.Nil(t, err)

	// no consumption from meter, return limit
	pwrAv = circ.GetRemainingPower()
	assert.Equal(t, limit, pwrAv)

	// set some consumption on meter
	mtr.pwr = 5000.0
	pwrAv = circ.GetRemainingPower()
	assert.Equal(t, limit-mtr.pwr, pwrAv)

	// simulate production in circuit (negative consumption)
	// available current is limit - consumption
	mtr.pwr = -5000.0
	pwrAv = circ.GetRemainingPower()
	assert.Equal(t, limit-mtr.pwr, pwrAv)
}

func TestHierarchyPower(t *testing.T) {
	// two circuits, check limit and consumption from both sides
	limitMain := 20000.0
	tstMtrMain := testMeter{pwr: 16000.0}
	circMain := NewCircuit(util.NewLogger("main"), 0, limitMain, nil, &tstMtrMain, &tstMtrMain)
	assert.NotNilf(t, circMain, "circuit not created")

	// add subcircuit with meter
	limitSub := 20000.0
	tstMtrSub := testMeter{pwr: 10000.0}
	circSub := NewCircuit(util.NewLogger("sub"), 0, limitSub, circMain, &tstMtrSub, &tstMtrSub)

	assert.NotNilf(t, circSub.parentCircuit, "parent circuit not set")
	assert.NotNilf(t, circSub.phaseCurrents, "sub circuit meter not set")

	curAv, err := circSub.CurrentPower()
	assert.Equal(t, curAv, 10000.0)
	assert.Nil(t, err)

	// remaining from sub circuit shall return the lower remaining from main
	assert.Equal(t, circMain.GetRemainingPower(), 4000.0)
	assert.Equal(t, circSub.GetRemainingPower(), 4000.0)

	// increasing the limit of main. Main has more remaining, sub limit is relevant
	circMain.maxPower = 30000.0
	assert.Equal(t, circMain.GetRemainingPower(), 14000.0)
	assert.Equal(t, circSub.GetRemainingPower(), 10000.0)
}

func TestNoCurrentMeter(t *testing.T) {
	// circuit may have nil for phase current
	tstMtr := testMeter{}
	circ := NewCircuit(util.NewLogger("main"), 0, 10, nil, nil, &tstMtr)
	assert.NotNilf(t, circ, "circuit not created")

	res := circ.GetRemainingCurrent()
	assert.Equal(t, math.MaxFloat64, res)
	assert.Nil(t, circ.update())
	_, err := circ.MaxPhasesCurrent()
	assert.NotNil(t, err)

	// we need a phase meter in casewe have a current limit
	circ = NewCircuit(util.NewLogger("main"), 10, 0, nil, nil, &tstMtr)
	assert.Nil(t, circ)

	// we always need a power meter
	circ = NewCircuit(util.NewLogger("main"), 0, 0, nil, nil, nil)
	assert.Nil(t, circ)

}
