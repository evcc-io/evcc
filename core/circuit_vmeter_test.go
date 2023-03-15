package core

import (
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/assert"
)

type testConsumerVM struct {
	cur float64
}

// interface consumer
func (tc *testConsumerVM) MaxPhasesCurrent() (float64, error) {
	return tc.cur, nil
}

func TestVMeter(t *testing.T) {
	// for testing we setup LP (consumers)

	vm := NewVMeter("test")

	var consumers []*testConsumerVM // internal access for testing
	for consId := 0; consId < 2; consId++ {
		cons := &testConsumerVM{}
		vm.Consumers = append(vm.Consumers, cons)
		consumers = append(consumers, cons)
	}

	var cur1, cur2, cur3 float64

	// no LP is consuming
	cur1, _, _, _ = vm.Currents()
	assert.Equal(t, cur1, 0.0)

	// one lp consumes current
	consumers[0].cur = maxA
	cur1, cur2, cur3, _ = vm.Currents()
	assert.Equal(t, cur1, maxA)
	// also check all 3 currents are identical
	assert.Equal(t, cur1, cur2)
	assert.Equal(t, cur1, cur3)

	// 2nd lp consumes current
	consumers[1].cur = 6.0
	cur1, _, _, _ = vm.Currents()
	assert.Equal(t, cur1, maxA+6.0)
}

func TestVMeterWithCircuit(t *testing.T) {
	// vmeter might use a circuit as consumer

	vm := NewVMeter("test")

	var consumers []*testConsumerVM // internal access for testing
	for consId := 0; consId < 2; consId++ {
		cons := &testConsumerVM{
			cur: maxA,
		}
		vm.Consumers = append(vm.Consumers, cons)
		consumers = append(consumers, cons)
	}
	// subcircuit
	testMeter := testMeter{cur: 10.0}
	circSub := NewCircuit(20.0, nil, &testMeter, util.NewLogger("test circuit Main"))
	assert.NotNilf(t, circSub, "circuit not created")
	vm.Consumers = append(vm.Consumers, circSub)

	var cur1 float64
	var cur2 float64
	var cur3 float64

	// expect to get the consumers current + circuit current
	cur1, cur2, cur3, _ = vm.Currents()
	assert.Equal(t, cur1, maxA*2+10.0)
	// also check all 3 currents are identical
	assert.Equal(t, cur1, cur2)
	assert.Equal(t, cur1, cur3)
}
