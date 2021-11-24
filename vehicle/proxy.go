package vehicle

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/templates"
	"gopkg.in/yaml.v3"
)

func init() {
	for _, tmpl := range templates.ByClass(templates.Vehicle) {
		instantiateFunc := instantiateFunction(tmpl)
		registry.Add(tmpl.Type(), instantiateFunc)
	}
}

func instantiateFunction(tmpl templates.Template) func(map[string]interface{}) (api.Vehicle, error) {
	return func(other map[string]interface{}) (api.Vehicle, error) {
		b, err := tmpl.RenderResult(false, other)
		if err != nil {
			return nil, err
		}

		var instance struct {
			Type  string
			Other map[string]interface{} `yaml:",inline"`
		}

		if err := yaml.Unmarshal(b, &instance); err != nil {
			return nil, err
		}

		return NewFromConfig(instance.Type, "", instance.Other)
	}
}
