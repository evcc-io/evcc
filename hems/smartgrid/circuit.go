package smartgrid

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

const GridControl = "gridcontrol"

// SetupCircuit returns or registers the grid control circuit
func SetupCircuit(title string) (api.Circuit, error) {
	// get root circuit
	root := circuit.Root()
	if root == nil {
		return nil, errors.New("hems requires load management- please configure root circuit")
	}

	if dev, err := config.Circuits().ByName(GridControl); err == nil {
		return dev.Instance(), err
	}

	// create new circuit
	circuit, err := circuit.New(util.NewLogger(GridControl), title, 0, 0, nil, time.Minute)
	if err != nil {
		return nil, err
	}

	dev := config.NewStaticDevice[api.Circuit](config.Named{Name: GridControl}, circuit)
	if err := config.Circuits().Add(dev); err != nil {
		return nil, err
	}

	// wrap old root with new grid control parent
	if err := root.Wrap(circuit); err != nil {
		return nil, err
	}

	return circuit, nil
}
