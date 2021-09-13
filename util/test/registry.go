package test

import (
	"bytes"
	"html/template"

	"github.com/evcc-io/config/registry"
	_ "github.com/evcc-io/config/templates" // import all config templates
	"gopkg.in/yaml.v3"
)

// ConfigTemplate is a configuration template from https://github.com/evcc-io/config
type ConfigTemplate struct {
	registry.Template
	Config map[string]interface{}
}

func ProcessSampleTemplate(tmpl registry.Template) registry.Template {
	sampleTmpl, err := template.New("sample").Parse(tmpl.Sample)
	if err != nil {
		// silently ignore errors here
		return tmpl
	}

	paramItems := make(map[string]interface{})

	for _, item := range tmpl.Params {
		paramItem := make(map[string]string)

		if item.Name == "" {
			// params name is required
			// silently ignore errors here
			return tmpl
		}
		if item.Value == "" && item.Type == "" {
			// params value or type is required
			// silently ignore errors here
			return tmpl
		}
		if item.Type != "" && len(item.Choice) == 0 {
			// params choice is required with type
			// silently ignore errors here
			return tmpl
		}

		if item.Value != "" {
			paramItem["value"] = item.Value
		}
		if item.Hint != "" {
			paramItem["hint"] = item.Hint
		}
		paramItems[item.Name] = paramItem
	}

	var tpl bytes.Buffer
	if err = sampleTmpl.Execute(&tpl, paramItems); err != nil {
		// silently ignore errors here
		return tmpl
	}

	tmpl.Sample = tpl.String()
	return tmpl
}

// ConfigTemplates returns configuration templates for giving class
func ConfigTemplates(class string) (res []ConfigTemplate) {
	templates := registry.TemplatesByClass(class)

	for _, tmpl := range templates {
		var conf map[string]interface{}
		template := ProcessSampleTemplate(tmpl)
		if err := yaml.Unmarshal([]byte(template.Sample), &conf); err != nil {
			// silently ignore errors here
			continue
		}

		parsed := ConfigTemplate{
			Template: template,
			Config:   conf,
		}

		res = append(res, parsed)
	}

	return res
}
