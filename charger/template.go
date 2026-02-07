package charger

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.AddCtx("template", NewChargerFromTemplateConfig)
}

func NewChargerFromTemplateConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	instance, err := templates.RenderInstance(templates.Charger, other)
	if err != nil {
		return nil, err
	}

	return NewFromConfig(ctx, instance.Type, instance.Other)
}
