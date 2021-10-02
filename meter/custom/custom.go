package custom

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"gopkg.in/yaml.v3"
)

type Registry interface {
	Add(name string, factory func(map[string]interface{}) (api.Meter, error))
}

type InstatiatorFunc func(typ string, other map[string]interface{}) (v api.Meter, err error)

func Register(registry Registry, instantiator InstatiatorFunc) {
	for _, tmpl := range templates {
		factory := templateFactory(tmpl, instantiator)
		registry.Add(tmpl.Type, factory)
	}
}

func templateFactory(tmpl Template, instantiator InstatiatorFunc) func(map[string]interface{}) (api.Meter, error) {
	params := make(map[string]Param)

	for _, p := range tmpl.Params {
		// add params to template for use as .Params.<ParamName>.<ParamAttribute>
		params[strings.Title(p.Name)] = p
	}

	return func(other map[string]interface{}) (api.Meter, error) {
		values := make(map[string]interface{})

		// set default values
		for _, p := range tmpl.Params {
			if p.Default != "" {
				values[p.Name] = p.Default
			}
		}

		if err := util.DecodeOther(other, &values); err != nil {
			return nil, err
		}

		t, err := template.New("yaml").Parse(tmpl.Render)
		// .Funcs(template.FuncMap(sprig.FuncMap())).Parse(tmpl.Sample)
		if err != nil {
			return nil, err
		}

		// make params available
		values["Params"] = params

		out := new(bytes.Buffer)
		if err := t.Execute(out, values); err != nil {
			return nil, err
		}

		fmt.Println("compiled template:\n", out.String())

		var instantiated struct {
			Type  string
			Other map[string]interface{} `yaml:",inline"`
		}

		if err := yaml.Unmarshal(out.Bytes(), &instantiated); err != nil {
			return nil, err
		}

		fmt.Println("parsed compilation:\n", instantiated)

		return instantiator(instantiated.Type, instantiated.Other)
	}
}
