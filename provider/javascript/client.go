package javascript

import (
	"github.com/evcc-io/evcc/util"
	"github.com/robertkrimen/otto"
)

// configure initializes JS VMs
func configure(other map[string]interface{}) error {
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
