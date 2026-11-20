package meter

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger/tuya"
	"github.com/evcc-io/evcc/util"
)

// Tuya smart plug meter implementation
func init() {
	registry.AddCtx("tuya", NewTuyaFromConfig)
}

// NewTuyaFromConfig creates a Tuya smart plug meter from generic config
func NewTuyaFromConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	var cc tuya.Config

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return tuya.NewPlug(ctx, cc)
}
