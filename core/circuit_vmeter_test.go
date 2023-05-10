package core

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

type testConsumerVM struct {
	cur    float64
	phases int
}

// interface consumer
func (tc *testConsumerVM) MaxPhasesCurrent() (float64, error) {
	return tc.cur, nil
}

func (tc *testConsumerVM) CurrentPower() (float64, error) {
	return CurrentToPower(tc.cur, uint(tc.phases)), nil
}

// TestVMeterWithConsumers tests with consumers
func TestVMeterWithConsumers(t *testing.T) {
	Voltage = 240
	vm := NewVMeter("test")

	for i := 0; i < 2; i++ {
		cons := new(testConsumerVM)
		cons.phases = 1
		vm.Consumers = append(vm.Consumers, cons)
	}

	var cur1, cur2, cur3, pwr float64

	// no LP is consuming
	cur1, _, _, _ = vm.Currents()
	assert.Equal(t, cur1, 0.0)

	// one lp consumes current
	vm.Consumers[0].(*testConsumerVM).cur = maxA
	cur1, cur2, cur3, _ = vm.Currents()
	assert.Equal(t, cur1, maxA)

	// also check all 3 currents are identical
	assert.Equal(t, cur1, cur2)
	assert.Equal(t, cur1, cur3)

	// check power
	pwr, _ = vm.CurrentPower()
	assert.Equal(t, CurrentToPower(cur1, 1), pwr)

	// 2nd lp consumes current
	vm.Consumers[1].(*testConsumerVM).cur = 6.0
	cur1, _, _, _ = vm.Currents()
	assert.Equal(t, cur1, maxA+6.0)
}

// TestVMeterWithCircuit tests with circuit as consumer (hierarchy)
func TestVMeterWithCircuit(t *testing.T) {
	Voltage = 240
	vm := NewVMeter("test") // meter under test

	for i := 0; i < 2; i++ {
		cons := &testConsumerVM{
			cur:    maxA,
			phases: 3,
		}
		vm.Consumers = append(vm.Consumers, cons)
	}

	// to remember correct power
	expectedPwr, _ := vm.CurrentPower()

	// subcircuit
	testMeter := testMeter{cur: 10.0}
	circSub := NewCircuit(util.NewLogger("foo"), 20.0, 0, nil, &testMeter, &testMeter)
	assert.NotNilf(t, circSub, "circuit not created")
	subPwr, _ := circSub.CurrentPower()
	expectedPwr = expectedPwr + subPwr
	vm.Consumers = append(vm.Consumers, circSub)

	var (
		l1 float64
		l2 float64
		l3 float64
	)

	// expect to get the consumers current + circuit current
	l1, l2, l3, _ = vm.Currents()
	assert.Equal(t, l1, maxA*2+10.0)
	// also check all 3 currents are identical
	assert.Equal(t, l1, l2)
	assert.Equal(t, l1, l3)

	// check the power
	pwr, _ := vm.CurrentPower()
	assert.Equal(t, pwr, expectedPwr)
}
