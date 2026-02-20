//go:build !linux

package plugin

import (
	"context"

	"github.com/evcc-io/evcc/api"
)

func init() {
	registry.AddCtx("gpio", NewGpioPluginFromConfig)
}

// NewGpioPluginFromConfig creates a GPIO provider
func NewGpioPluginFromConfig(ctx context.Context, other map[string]any) (Plugin, error) {
	return nil, api.ErrUnsupportedPlatform
}
