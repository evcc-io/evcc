//go:build !linux

package charger

import (
	"context"

	"github.com/evcc-io/evcc/api"
)

func init() {
	registry.AddCtx("openwb-native", NewOpenWbNativeFromConfig)
}

// NewOpenWbNativeFromConfig creates an OpenWbNative DIN charger from generic config
func NewOpenWbNativeFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	return nil, api.ErrUnsupportedPlatform
}
