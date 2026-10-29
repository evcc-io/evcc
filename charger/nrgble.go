//go:build !linux

package charger

import (
	"context"
	"errors"

	"github.com/evcc-io/evcc/api"
)

func init() {
	registry.AddCtx("nrgkick-bluetooth", NewNRGKickBLEFromConfig)
}

// NewNRGKickBLEFromConfig creates a NRGKickBLE charger from generic config
func NewNRGKickBLEFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	return nil, errors.New("NRGKick bluetooth is only supported on linux")
}
