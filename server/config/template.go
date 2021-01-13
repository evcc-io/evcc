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

func configFromTemplate(tmpl templates.Template) (configType, error) {
	res := configType{
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
	desc := TypeDefinition(tmpl.Class, tmpl.Type)

	// unexported or not found
	if desc.Type == "" || desc.Config == nil {
		return res, fmt.Errorf("type description not available for: %s of type %s:%s", tmpl.Name, tmpl.Class, tmpl.Type)
	}

	// unmarshal template into config type description
	if err = util.DecodeOther(conf, &desc.Config); err != nil {
		return res, err
	}

	res.Fields = Describe(desc.Config)

	return res, nil
}

// Templates returns configuration templates for giving class
func Templates(class string) []configType {
	types := templates.TemplatesByClass(class)

	// name -> type
	sort.Slice(types, func(i, j int) bool {
		if types[i].Name < types[j].Name {
			return true
		}
		return strings.Compare(types[i].Type, types[j].Type) < 0
	})

	res := make([]configType, 0, len(types))

	for _, tmpl := range types {
		ct, err := configFromTemplate(tmpl)
		if err != nil {
			continue
		}

		res = append(res, ct)
	}

	return res
}
