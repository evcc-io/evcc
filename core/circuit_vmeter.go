// virtual meter which evaluates current based on conneced consumers
package core

import (
	"github.com/evcc-io/evcc/util"
)

// interface to get the current in use from a consumer
// it is expected to get the max current over all phases
type Consumer interface {
	MaxPhasesCurrent() (float64, error)
}

type VMeter struct {
	log       *util.Logger
	Consumers []Consumer // all consumers under management. Used for consumption evaluation
}

// AddConsumer adds a consumer to evaluate consumption
func (vm *VMeter) AddConsumer(c Consumer) {
	vm.Consumers = append(vm.Consumers, c)
}

// Currents implements MeterCurrent interface
// return current as it would be used on all 3 phases. We dont phase accurate evaluation for the installation.
func (vm *VMeter) Currents() (float64, float64, float64, error) {
	vm.log.TRACE.Printf("get current from %d consumers", len(vm.Consumers))
	var currentTotal float64
	for _, consumer := range vm.Consumers {
		if cur, err := consumer.MaxPhasesCurrent(); err == nil {
			vm.log.TRACE.Printf("add %.1fA current from consumer", cur)
			currentTotal += cur
		} else {
			return 0, 0, 0, err
		}
	}
	return currentTotal, currentTotal, currentTotal, nil
}

// NewVMeter a new vmeter
func NewVMeter(n string) *VMeter {
	vm := &VMeter{
		log: util.NewLogger("vmtr-" + n),
	}
	return vm
}
