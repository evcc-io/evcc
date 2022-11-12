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

	var res api.Vehicle
	if err == nil {
		res, err = NewFromConfig(instance.Type, instance.Other)
	}

	return res, err
}
