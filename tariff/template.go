package tariff

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.Add("template", NewTariffFromTemplateConfig)
}

func NewTariffFromTemplateConfig(other map[string]interface{}) (api.Tariff, error) {
	instance, err := templates.RenderInstance(templates.Tariff, other)
	if err != nil {
		return nil, err
	}

	return NewFromConfig(instance.Type, instance.Other)
}
