package charger

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/templates"
	"gopkg.in/yaml.v3"
)

func init() {
	registry.Add("template", NewChargerFromTemplateConfig)
}

func NewChargerFromTemplateConfig(other map[string]interface{}) (api.Charger, error) {
	name := other["template"].(string)
	if name == "" {
		return nil, fmt.Errorf("missing template name")
	}
	tmpl, err := templates.ByTemplate(name, templates.Charger)
	if err != nil {
		return nil, err
	}

	b, _, err := tmpl.RenderResult(false, other)
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

	return NewFromConfig(instance.Type, instance.Other)
}
