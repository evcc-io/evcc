package tempsensor

import (
	"context"
	"errors"
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/plugin"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.AddCtx(api.Custom, NewConfigurableFromConfig)
}

// NewConfigurableFromConfig creates a temp sensor from config
func NewConfigurableFromConfig(ctx context.Context, other map[string]any) (api.Battery, error) {
	cc := struct {
		Temp *plugin.Config
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Temp == nil {
		return nil, errors.New("missing temp")
	}

	tempG, err := cc.Temp.FloatGetter(ctx)
	if err != nil {
		return nil, fmt.Errorf("temp: %w", err)
	}

	return implement.Battery(tempG), nil
}
