package charger

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
)

func init() {
	registry.AddCtx("device", NewDeviceFromConfig)
}

// NewDeviceFromConfig returns an already configured charger by name
func NewDeviceFromConfig(_ context.Context, other map[string]any) (api.Charger, error) {
	var cc struct{ Name string }
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return config.Instance(config.Chargers(), cc.Name)
}
