package templates

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/util"
)

const (
	ParamUsage = "usage"
)

type Param struct {
	Name    string
	Default string
	Hint    string
	Choice  []string
}

type Template struct {
	Type   string
	Params []Param
	Render string // rendering template
}

func (t *Template) Defaults() map[string]interface{} {
	values := make(map[string]interface{})
	for _, p := range t.Params {
		if p.Default != "" {
			values[p.Name] = p.Default
		}
	}

	return values
}

func (t *Template) Usages() []string {
	for _, p := range t.Params {
		if p.Name == ParamUsage {
			return p.Choice
		}
	}

	return nil
}

//go:embed proxy.tpl
var proxyTmpl string

func (t *Template) RenderProxy() ([]byte, error) {
	tmpl, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(proxyTmpl)
	if err != nil {
		panic(err)
	}

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, map[string]interface{}{
		"Type":   t.Type,
		"Params": t.Params,
	})

	return bytes.TrimSpace(out.Bytes()), err
}

func (t *Template) RenderResult(other map[string]interface{}) ([]byte, error) {
	values := t.Defaults()
	if err := util.DecodeOther(other, &values); err != nil {
		return nil, err
	}

	tmpl, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(t.Render)
	if err != nil {
		return nil, err
	}

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, values)

	return bytes.TrimSpace(out.Bytes()), err
}
