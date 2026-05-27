package circuit

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.AddCtx("template", NewFromTemplateConfig)
}

func NewFromTemplateConfig(ctx context.Context, other map[string]any) (api.Circuit, error) {
	instance, err := templates.RenderInstance(templates.Circuit, other)
	if err != nil {
		return nil, err
	}

	return NewConfigurableFromConfig(ctx, instance.Other)
}
