package meter

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/ecoflow"
)

func init() {
	registry.AddCtx("ecoflow-stream", func(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
		return ecoflow.NewEcoFlowStreamFromConfig(ctx, other)
	})
	registry.AddCtx("ecoflow-powerstream", func(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
		return ecoflow.NewEcoFlowPowerStreamFromConfig(ctx, other)
	})
}
