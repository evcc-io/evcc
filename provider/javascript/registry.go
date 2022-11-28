package javascript

import (
	"sync"

	"github.com/robertkrimen/otto"
	_ "github.com/robertkrimen/otto/underscore"
)

var (
	mu       sync.Mutex
	registry = make(map[string]*otto.Otto)
)

// RegisteredVM returns a JS VM. If name is not empty, it will return a shared instance.
func RegisteredVM(name, init string) (*otto.Otto, error) {
	mu.Lock()
	defer mu.Unlock()

	vm, ok := registry[name]

	// create new VM
	if !ok {
		vm = otto.New()

		if init != "" {
			if _, err := vm.Run(init); err != nil {
				return nil, err
			}
		}

		if name != "" {
			registry[name] = vm
		}
	}

	return vm, nil
}
