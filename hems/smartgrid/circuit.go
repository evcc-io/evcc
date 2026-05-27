package smartgrid

import (
	"errors"
	"sync"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

const GridControl = "gridcontrol"

var (
	mu      sync.Mutex
	managed api.Circuit
)

// SetupCircuit returns or registers the grid control circuit.
// Subsequent calls return the previously registered instance (idempotent).
func SetupCircuit() (api.Circuit, error) {
	mu.Lock()
	defer mu.Unlock()

	if managed != nil {
		return managed, nil
	}

	if _, err := config.Circuits().ByName(GridControl); err == nil {
		return nil, errors.New("gridcontrol is a reserved name and will be auto-created as root circuit when hems is configured")
	}

	root := circuit.Root()

	c, err := circuit.New(util.NewLogger(GridControl), "", 0, 0, nil, time.Minute)
	if err != nil {
		return nil, err
	}

	dev := config.NewStaticDevice[api.Circuit](config.Named{Name: GridControl}, c)
	if err := config.Circuits().Add(dev); err != nil {
		return nil, err
	}

	// wrap old root with new grid control parent
	if root != nil {
		if err := root.Wrap(c); err != nil {
			return nil, err
		}
	}

	managed = c
	return c, nil
}
