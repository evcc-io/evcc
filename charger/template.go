package charger

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.Add("template", NewChargerFromTemplateConfig)
}

func NewChargerFromTemplateConfig(other map[string]interface{}) (api.Charger, error) {
	instance, err := templates.RenderInstance(templates.Charger, other)
	if err != nil {
		return nil, err
	}

	return NewFromConfig(instance.Type, instance.Other)
}
