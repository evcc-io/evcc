package config

import (
	"fmt"
	"sort"
	"strings"

	templates "github.com/andig/evcc-config/registry"
	_ "github.com/andig/evcc-config/templates" // import all config templates
	"github.com/andig/evcc/util"

	"gopkg.in/yaml.v3"
)

// configFromTemplate converts configuration template to annotated configuration type
func configFromTemplate(tmpl templates.Template) (description, error) {
	res := description{
		Type:  tmpl.Type,
		Label: tmpl.Name,
	}

	// template yaml to map
	var conf map[string]interface{}
	err := yaml.Unmarshal([]byte(tmpl.Sample), &conf)
	if err != nil {
		return res, err
	}

	// get matching config type description for template type
	desc, err := typeDefinition(tmpl.Class, tmpl.Type)
	if err != nil {
		return res, err
	}

	// unexported or not found
	if desc.Type == "" || desc.Config == nil {
		return res, fmt.Errorf("type description not available for: %s of type %s:%s", tmpl.Name, tmpl.Class, tmpl.Type)
	}

	// unmarshal template into config type description
	if err = util.DecodeOther(conf, &desc.Config); err != nil {
		return res, err
	}

	res.Fields = prependType(tmpl.Type, annotate(desc.Config))

	return res, nil
}

// Templates returns configuration templates for giving class
func Templates(class string) []interface{} {
	types := templates.TemplatesByClass(class)

	// name -> type
	sort.Slice(types, func(i, j int) bool {
		if types[i].Name < types[j].Name {
			return true
		}
		return strings.ToLower(types[i].Type) < strings.ToLower(types[j].Type)
	})

	res := make([]interface{}, 0, len(types))

	for _, tmpl := range types {
		ct, err := configFromTemplate(tmpl)
		if err != nil {
			continue
		}

		res = append(res, ct)
	}

	return res
}
