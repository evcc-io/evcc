package custom

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig"
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
		println("")
		println(strings.ToUpper(tmpl.Type))
		buildSample(tmpl)

		factory := renderFactory(tmpl, instantiator)
		registry.Add(tmpl.Type, factory)
	}
}

var sampleTmpl = `
{{- range . -}}
{{ .Name }}:
	{{- if len .Choice }} {{ join "|" .Choice }} {{- else }} {{ .Default }} {{- end }}
	{{- if .Hint }} # {{ .Hint }} {{- end }}
{{ end -}}
`

func buildSample(tmpl Template) {
	t, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(sampleTmpl)
	if err != nil {
		panic(err)
	}

	out := new(bytes.Buffer)
	if err := t.Execute(out, tmpl.Params); err != nil {
		panic(err)
	}

	println("sample:")
	println(out.String())
}

func renderFactory(tmpl Template, instantiator InstatiatorFunc) func(map[string]interface{}) (api.Meter, error) {
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

		t, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(tmpl.Render)
		if err != nil {
			return nil, err
		}

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
