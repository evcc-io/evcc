package hems

import (
	"context"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/hems"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.AddCtx("template", NewHemsFromTemplateConfig)
}

func NewHemsFromTemplateConfig(ctx context.Context, other map[string]any, site site.API) (hems.API, error) {
	instance, err := templates.RenderInstance(templates.Hems, other)
	if err != nil {
		return nil, err
	}

	return NewFromConfig(ctx, instance.Type, instance.Other, site)
}
