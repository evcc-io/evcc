package shared

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

// GetOrCreateCircuit returns or registers a circuit if not already registered
func GetOrCreateCircuit(name, title string) (api.Circuit, error) {
	if dev, err := config.Circuits().ByName(name); err == nil {
		return dev.Instance(), err
	}

	// create new circuit
	circuit, err := circuit.New(util.NewLogger(name), title, 0, 0, nil, time.Minute)
	if err != nil {
		return nil, err
	}

	dev := config.NewStaticDevice[api.Circuit](config.Named{Name: name}, circuit)
	if err := config.Circuits().Add(dev); err != nil {
		return nil, err
	}

	return circuit, nil
}
