package meter

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.AddCtx("template", NewMeterFromTemplateConfig)
}

func NewMeterFromTemplateConfig(ctx context.Context, other map[string]interface{}) (api.Meter, error) {
	instance, err := templates.RenderInstance(templates.Meter, other)
	if err != nil {
		return nil, err
	}

	return NewFromConfig(ctx, instance.Type, instance.Other)
}
