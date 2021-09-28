package test

import (
	"github.com/evcc-io/config/registry"
	_ "github.com/evcc-io/config/templates" // import all config templates
	"gopkg.in/yaml.v3"
)

// ConfigTemplate is a configuration template from https://github.com/evcc-io/config
type ConfigTemplate struct {
	registry.Template
	Config map[string]interface{}
}

// ConfigTemplates returns configuration templates for giving class
func ConfigTemplates(class string) (res []ConfigTemplate) {
	templates := registry.TemplatesByClass(class)

	for _, tmpl := range templates {
		var conf map[string]interface{}
		if err := yaml.Unmarshal([]byte(tmpl.Sample), &conf); err != nil {
			// silently ignore errors here
			continue
		}

		parsed := ConfigTemplate{
			Template: tmpl,
			Config:   conf,
		}

		res = append(res, parsed)
	}

	return res
}
