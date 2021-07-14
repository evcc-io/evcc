package wrapper

import (
	"fmt"

	"github.com/andig/evcc/api"
)

// Wrapper wraps an api.Vehicle to capture initialization errors
type Wrapper struct {
	api.Vehicle
	err error
}

// New creates a new Vehicle
func New(w api.Vehicle, err error) (api.Vehicle, error) {
	v := &Wrapper{
		err:     fmt.Errorf("vehicle not available: %w", err),
		Vehicle: w,
	}

	return v, nil
}

// SoC implements the api.Vehicle interface
func (v *Wrapper) SoC() (float64, error) {
	if v.err != nil {
		return 0, v.err
	}

	return v.Vehicle.SoC()
}
