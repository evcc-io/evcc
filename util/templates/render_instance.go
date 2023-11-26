package templates

import (
	"errors"

	"github.com/evcc-io/evcc/util"
	"gopkg.in/yaml.v3"
)

// Instance is an actual instantiated template
type Instance struct {
	Type  string
	Other map[string]interface{} `yaml:",inline"`
}

// RenderInstance renders an actual configuration instance
func RenderInstance(class Class, other map[string]interface{}) (*Instance, error) {
	var cc struct {
		Template string
		Other    map[string]interface{} `mapstructure:",remain"`
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	tmpl, err := ByName(class, cc.Template)
	if err != nil {
		return nil, err
	}

	b, _, err := tmpl.RenderResult(TemplateRenderModeInstance, other)
	if err != nil {
		return nil, err
	}

	var instance Instance
	if err = yaml.Unmarshal(b, &instance); err == nil && instance.Type == "" {
		err = errors.New("empty instance type- check for missing usage")
	}

	return &instance, err
}
