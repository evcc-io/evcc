package golang

import (
	"strings"
	"sync"

	"github.com/evcc-io/evcc/plugin/golang/stdlib"
	"github.com/traefik/yaegi/interp"
)

// VM is an interpreter with the lock serializing its execution
type VM struct {
	*interp.Interpreter
	sync.Mutex
}

var (
	mu       sync.Mutex
	registry = make(map[string]*VM)
)

// RegisteredVM returns a Go VM. If name is not empty, it will return a shared instance.
func RegisteredVM(name, init string) (*VM, error) {
	mu.Lock()
	defer mu.Unlock()

	name = strings.ToLower(name)
	vm, ok := registry[name]

	// create new VM
	if !ok {
		vm = &VM{Interpreter: interp.New(interp.Options{})}
		if err := vm.Use(stdlib.Symbols); err != nil {
			return nil, err
		}
		vm.ImportUsed()

		if init != "" {
			if _, err := vm.Eval(init); err != nil {
				return nil, err
			}
		}

		if name != "" {
			registry[name] = vm
		}
	}

	return vm, nil
}
