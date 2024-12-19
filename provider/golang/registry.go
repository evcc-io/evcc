package golang

import (
	"strings"
	"sync"

	"github.com/evcc-io/evcc/provider/golang/stdlib"
	"github.com/traefik/yaegi/interp"
)

const Imports = `import (
	"fmt"
	"math"
	"strings"
	"time"
)`

var (
	mu       sync.Mutex
	registry = make(map[string]*interp.Interpreter)
)

// RegisteredVM returns a JS VM. If name is not empty, it will return a shared instance.
func RegisteredVM(name, init string) (*interp.Interpreter, error) {
	mu.Lock()
	defer mu.Unlock()

	name = strings.ToLower(name)
	vm, ok := registry[name]

	// create new VM
	if !ok {
		vm = interp.New(interp.Options{})

		if err := vm.Use(stdlib.Symbols); err != nil {
			return nil, err
		}

		if _, err := vm.Eval(Imports); err != nil {
			return nil, err
		}

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
