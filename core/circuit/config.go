package circuit

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
)

func Root() api.Circuit {
	for _, dev := range config.Circuits().Devices() {
		if c := dev.Instance(); c.GetParent() == nil {
			return c
		}
	}
	return nil
}
