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
	vm := NewVMeter("test")

	for i := 0; i < 2; i++ {
		cons := new(testConsumerVM)
		vm.Consumers = append(vm.Consumers, cons)
	}

	var cur1, cur2, cur3 float64

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

	// 2nd lp consumes current
	vm.Consumers[1].(*testConsumerVM).cur = 6.0
	cur1, _, _, _ = vm.Currents()
	assert.Equal(t, cur1, maxA+6.0)
}

func TestVMeterWithCircuit(t *testing.T) {
	vm := NewVMeter("test")

	for i := 0; i < 2; i++ {
		cons := &testConsumerVM{
			cur: maxA,
		}
		vm.Consumers = append(vm.Consumers, cons)
	}

	// subcircuit
	testMeter := testMeter{cur: 10.0}
	circSub := NewCircuit(util.NewLogger("foo"), 20.0, nil, &testMeter)
	assert.NotNilf(t, circSub, "circuit not created")
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
}
