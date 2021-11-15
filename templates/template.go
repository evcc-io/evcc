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

const (
	ParamValueTypeString = "string"
	ParamValueTypeNumber = "number"
	ParamValueTypeFloat  = "float"
	ParamValueTypeBool   = "bool"
)

var ParamValueTypes = []string{ParamValueTypeString, ParamValueTypeNumber, ParamValueTypeBool}

// Template describes is a proxy device for use with cli and automated testing
type Template struct {
	Type         string
	Description  string // user friendly description of the device this template describes
	Requirements Requirements
	GuidedSetup  GuidedSetup
	Generic      bool // if this describes a generic device type rather than a product
	Params       []Param
	Render       string // rendering template
}

// Requirements
type Requirements struct {
	Eebus       bool   // EEBUS Setup is required
	Sponsorship bool   // Sponsorship is required
	Description string // Description of requirements, e.g. how the device needs to be prepared
	URI         string // URI to a webpage with more details about the preparation requirements
}

type GuidedSetup struct {
	Enable bool             // if true, guided setup is possible
	Linked []LinkedTemplate // a list of templates that should be processed as part of the guided setup
}

// Linked Template
type LinkedTemplate struct {
	Type  string
	Usage string // usage: "grid", "pv", "battery"
}

// Param is a proxy template parameter
type Param struct {
	Name      string
	Required  bool     // cli if the user has to provide a non empty value
	Mask      bool     // cli if the value should be masked, e.g. for passwords
	Advanced  bool     // cli if the user does not need to be asked. Requires a "Default" to be defined.
	Default   string   // default value if no user value is provided in the configuration
	Example   string   // cli example value
	Help      string   // cli configuration help
	Test      string   // testing default value
	Value     string   // user provided value via cli configuration
	ValueType string   // string representation of the value type, "string" is default
	Choice    []string // defines which usage choices this config supports, valid elemtents are "grid", "pv", "battery", "charge"
	Usages    []string
}

// Defaults returns a map of default values for the template
func (t *Template) Defaults(docs bool) map[string]interface{} {
	values := make(map[string]interface{})
	for _, p := range t.Params {
		if p.Test != "" {
			values[p.Name] = p.Test
		} else if p.Example != "" && docs {
			values[p.Name] = p.Example
		} else {
			values[p.Name] = p.Default // may be empty
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
	return t.RenderProxyWithValues(nil)
}

func (t *Template) RenderProxyWithValues(values map[string]interface{}) ([]byte, error) {
	tmpl, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(proxyTmpl)
	if err != nil {
		panic(err)
	}

	for index, p := range t.Params {
		for k, v := range values {
			if p.Name == k {
				t.Params[index].Value = v.(string)
			}
		}
	}

	// remove params with no values, no defaults and no example
	var newParams []Param
	for _, param := range t.Params {
		if param.Value == "" && param.Default == "" && param.Example == "" && !param.Required {
			continue
		}
		newParams = append(newParams, param)
	}
	t.Params = newParams

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, map[string]interface{}{
		"Template": t.Type,
		"Params":   t.Params,
	})

	return bytes.TrimSpace(out.Bytes()), err
}

// RenderResult renders the result template to instantiate the proxy
func (t *Template) RenderResult(docs bool, other map[string]interface{}) ([]byte, error) {
	values := t.Defaults(docs)
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
