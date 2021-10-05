package templates

import (
	"bytes"
	_ "embed"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/util"
)

const (
	ParamUsage         = "usage"
	ParamModbus        = "modbus"
	ModbusMagicComment = "# ::modbus-setup::"
)

// Template describes is a proxy device for use with cli and automated testing
type Template struct {
	Type   string
	Params []Param
	Render string // rendering template
}

// Param is a proxy template parameter
type Param struct {
	Name    string
	Default string // cli configuration default value
	Hint    string // cli configuration hint
	Test    string // testing default value
	Choice  []string
	Usages  []string
}

// Defaults returns a map of default values for the template
func (t *Template) Defaults() map[string]interface{} {
	values := make(map[string]interface{})
	for _, p := range t.Params {
		if p.Test != "" {
			values[p.Name] = p.Test
		} else if p.Default != "" {
			values[p.Name] = p.Default
		} else {
			values[p.Name] = ""
		}
	}

	return values
}

// Usages returns the list of supported usages
func (t *Template) Usages() []string {
	for _, p := range t.Params {
		if p.Name == ParamUsage {
			return p.Choice
		}
	}

	return nil
}

func (t *Template) ModbusChoices() []string {
	for _, p := range t.Params {
		if p.Name == ParamModbus {
			return p.Choice
		}
	}

	return nil
}

//go:embed proxy.tpl
var proxyTmpl string

// RenderProxy renders the proxy template for inclusion in documentation
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

// RenderResult renders the result template to instantiate the proxy
func (t *Template) RenderResult(other map[string]interface{}) ([]byte, error) {
	values := t.Defaults()
	if err := util.DecodeOther(other, &values); err != nil {
		return nil, err
	}

	if err := t.RenderModbus(values); err != nil {
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
