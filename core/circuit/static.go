package circuit

import (
	"github.com/evcc-io/evcc/api"
)

func init() {
	registry.Add("static", NewStaticCircuitFromConfig)
}

// NewStaticCircuitFromConfig creates new static circuit
func NewStaticCircuitFromConfig(other map[string]any) (api.Circuit, error) {
	return NewFromConfig(other)
}
