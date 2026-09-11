package thermometer

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.AddCtx("template", NewFromTemplateConfig)
}

func NewFromTemplateConfig(ctx context.Context, other map[string]any) (api.Battery, error) {
	instance, err := templates.RenderInstance(templates.Thermometer, other)
	if err != nil {
		return nil, err
	}

	return NewFromConfig(ctx, instance.Type, instance.Other)
}
