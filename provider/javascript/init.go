package javascript

import (
	"github.com/andig/evcc/util"
	"github.com/robertkrimen/otto"
)

// Configure initializes JS VMs
func Configure(other map[string]interface{}) error {
	cc := []struct {
		VM     string
		Script string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return err
	}

	// init all VMs that require it
	for _, conf := range cc {
		if conf.Script == "" {
			continue
		}

		if _, ok := registry[conf.VM]; !ok {
			vm := otto.New()

			_, err := vm.Run(conf.Script)
			if err != nil {
				return err
			}

			registry[conf.VM] = vm
		}
	}

	return nil
}

var registry = make(map[string]*otto.Otto)

// RegisteredVM returns a JS VM. If name is not empty, it will return a shared instance.
func RegisteredVM(name string) *otto.Otto {
	vm, ok := registry[name]

	// create new VM
	if !ok {
		vm = otto.New()

		if name != "" {
			registry[name] = vm
		}
	}

	return vm
}
