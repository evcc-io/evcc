package tariff

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.AddCtx("template", NewTariffFromTemplateConfig)
}

func NewTariffFromTemplateConfig(ctx context.Context, other map[string]interface{}) (api.Tariff, error) {
	instance, err := templates.RenderInstance(templates.Tariff, other)
	if err != nil {
		return nil, err
	}

	return NewFromConfig(ctx, instance.Type, instance.Other)
}
