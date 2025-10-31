//go:build !linux

package charger

import (
	"errors"

	"github.com/evcc-io/evcc/api"
)

func init() {
	registry.Add("nrgkick-bluetooth", NewNRGKickBLEFromConfig)
}

// NewNRGKickBLEFromConfig creates a NRGKickBLE charger from generic config
func NewNRGKickBLEFromConfig(other map[string]any) (api.Charger, error) {
	return nil, errors.New("NRGKick bluetooth is only supported on linux")
}
