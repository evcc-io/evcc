package vehicle

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	registry.Add("template", NewVehicleFromTemplateConfig)
}

func NewVehicleFromTemplateConfig(other map[string]interface{}) (api.Vehicle, error) {
	instance, err := templates.RenderInstance(templates.Vehicle, other)
	if err != nil {
		return nil, err
	}

	return NewFromConfig(instance.Type, instance.Other)
}
