package test

import (
	"github.com/andig/evcc-config/registry"
	_ "github.com/andig/evcc-config/templates"
	"gopkg.in/yaml.v3"
)

type ParsedTempalte struct {
	registry.Template
	Config map[string]interface{}
}

// ConfigTemplates returns configuration templates for giving class
func ConfigTemplates(class string) (res []ParsedTempalte) {
	templates := registry.TemplatesByClass(class)

	for _, tmpl := range templates {
		var conf map[string]interface{}
		if err := yaml.Unmarshal([]byte(tmpl.Sample), &conf); err != nil {
			// silently ignore errors here
			continue
		}

		parsed := ParsedTempalte{
			Template: tmpl,
			Config:   conf,
		}

		res = append(res, parsed)
	}

	return res
}
