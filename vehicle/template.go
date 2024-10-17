package vehicle

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.AddCtx("template", NewVehicleFromTemplateConfig)
}

func NewVehicleFromTemplateConfig(ctx context.Context, other map[string]interface{}) (api.Vehicle, error) {
	instance, err := templates.RenderInstance(templates.Vehicle, other)
	if err != nil {
		return nil, err
	}

	return NewFromConfig(ctx, instance.Type, instance.Other)
}
