package wrapper

import (
	"fmt"

	"github.com/andig/evcc/api"
)

// Wrapper wraps an api.Vehicle to capture initialization errors
type Wrapper struct {
	err error
}

// New creates a new Vehicle
func New(w api.Vehicle, err error) (api.Vehicle, error) {
	v := &Wrapper{
		err: fmt.Errorf("vehicle not available: %w", err),
	}

	return v, nil
}

// Title implements the Vehicle.Title interface
func (v *Wrapper) Title() string {
	return "unavailable"
}

// Capacity implements the Vehicle.Capacity interface
func (v *Wrapper) Capacity() int64 {
	return 0
}

// Identify implements the api.Identifier interface
func (v *Wrapper) Identify() (string, error) {
	return "", v.err
}

// SoC implements the api.Vehicle interface
func (v *Wrapper) SoC() (float64, error) {
	return 0, v.err
}
