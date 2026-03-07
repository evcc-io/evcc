package meter

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/spf13/cast"
)

func init() {
	registry.AddCtx("template", NewFromTemplateConfig)
}

func NewFromTemplateConfig(ctx context.Context, other map[string]any) (api.Meter, error) {
	reversed := cast.ToBool(other["reversed"])

	instance, err := templates.RenderInstance(templates.Meter, other)
	if err != nil {
		return nil, err
	}

	if reversed {
		if instance.Other == nil {
			instance.Other = make(map[string]any)
		}
		instance.Other["reversed"] = true
	}

	return NewFromConfig(ctx, instance.Type, instance.Other)
}
