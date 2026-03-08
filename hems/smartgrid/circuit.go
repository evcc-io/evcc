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
func SetupCircuit() (api.Circuit, error) {
	if _, err := config.Circuits().ByName(GridControl); err == nil {
		return nil, errors.New("gridcontrol is a reserved name and will be auto-created as root circuit when hems is configured")
	}

	root := circuit.Root()

	// create new circuit
	circuit, err := circuit.New(util.NewLogger(GridControl), "", 0, 0, nil, time.Minute)
	if err != nil {
		return nil, err
	}

	dev := config.NewStaticDevice[api.Circuit](config.Named{Name: GridControl}, circuit)
	if err := config.Circuits().Add(dev); err != nil {
		return nil, err
	}

	// wrap old root with new grid control parent
	if root != nil {
		if err := root.Wrap(circuit); err != nil {
			return nil, err
		}
	}

	return circuit, nil
}
