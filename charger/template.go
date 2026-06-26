package charger

import (
	"context"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.AddCtx("template", NewFromTemplateConfig)
}

func NewFromTemplateConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	instance, err := templates.RenderInstance(templates.Charger, other)
	if err != nil {
		return nil, err
	}

	c, err := NewFromConfig(ctx, instance.Type, instance.Other)
	if err != nil {
		return nil, err
	}

	// Allow the charger to clean up deprecated params from the outer template-level
	// config map (e.g. credentials that have been migrated to persistent storage).
	if cleaner, ok := c.(templateConfigCleaner); ok {
		cleaner.cleanTemplateConfig(other)
	}

	return c, nil
}

// templateConfigCleaner is an optional interface that a charger may implement to
// clean up template-level config params after successful creation. The map passed
// to cleanTemplateConfig is the outer template-param map stored in the database.
type templateConfigCleaner interface {
	cleanTemplateConfig(map[string]any)
}
