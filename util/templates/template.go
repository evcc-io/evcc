package templates

import (
	"bytes"
	_ "embed"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/templates/definition"
	"github.com/evcc-io/evcc/util"
	"gopkg.in/yaml.v3"
)

const (
	ParamUsage  = "usage"
	ParamModbus = "modbus"

	HemsTypeSMA = "sma"

	ModbusChoiceRS485    = "rs485"
	ModbusChoiceTCPIP    = "tcpip"
	ModbusKeyRS485Serial = "rs485serial"
	ModbusKeyRS485TCPIP  = "rs485tcpip"
	ModbusKeyTCPIP       = "tcpip"

	ModbusRS485Serial = "modbusrs485serial"
	ModbusRS485TCPIP  = "modbusrs485tcpip"
	ModbusTCPIP       = "modbustcpip"

	ModbusParamNameId        = "id"
	ModbusParamValueId       = 1
	ModbusParamNameDevice    = "device"
	ModbusParamValueDevice   = "/dev/ttyUSB0"
	ModbusParamNameBaudrate  = "baudrate"
	ModbusParamValueBaudrate = 9600
	ModbusParamNameComset    = "comset"
	ModbusParamValueComset   = "8N1"
	ModbusParamNameURI       = "uri"
	ModbusParamNameHost      = "host"
	ModbusParamValueHost     = "192.0.2.2"
	ModbusParamNamePort      = "port"
	ModbusParamValuePort     = 502
	ModbusParamNameRTU       = "rtu"
)

var HemsValueTypes = []string{HemsTypeSMA}

const (
	ParamValueTypeString     = "string"
	ParamValueTypeNumber     = "number"
	ParamValueTypeFloat      = "float"
	ParamValueTypeBool       = "bool"
	ParamValueTypeStringList = "stringlist"
)

var ParamValueTypes = []string{ParamValueTypeString, ParamValueTypeNumber, ParamValueTypeBool}

// language specific texts
type TextLanguage struct {
	DE string // german text
	EN string // english text
}

func (t *TextLanguage) String(lang string) string {
	switch lang {
	case "de":
		return t.DE
	case "en":
		return t.EN
	}
	return t.DE
}

func (t *TextLanguage) SetString(lang, value string) {
	switch lang {
	case "de":
		t.DE = value
	case "en":
		t.EN = value
	default:
		t.DE = value
	}
}

// Requirements
type Requirements struct {
	Hems        string       // HEMS Type
	Eebus       bool         // EEBUS Setup is required
	Sponsorship bool         // Sponsorship is required
	Description TextLanguage // Description of requirements, e.g. how the device needs to be prepared
	URI         string       // URI to a webpage with more details about the preparation requirements
}

type GuidedSetup struct {
	Enable bool             // if true, guided setup is possible
	Linked []LinkedTemplate // a list of templates that should be processed as part of the guided setup
}

// Linked Template
type LinkedTemplate struct {
	Template        string
	Usage           string // usage: "grid", "pv", "battery"
	Multiple        bool   // if true, multiple instances of this template can be added
	ExcludeTemplate string // only consider this if no device of the named linked template was added
}

// Param is a proxy template parameter
type Param struct {
	Name      string
	Required  bool         // cli if the user has to provide a non empty value
	Mask      bool         // cli if the value should be masked, e.g. for passwords
	Advanced  bool         // cli if the user does not need to be asked. Requires a "Default" to be defined.
	Default   string       // default value if no user value is provided in the configuration
	Example   string       // cli example value
	Help      TextLanguage // cli configuration help
	Test      string       // testing default value
	Value     string       // user provided value via cli configuration
	Values    []string     // user provided list of values
	ValueType string       // string representation of the value type, "string" is default
	Choice    []string     // defines which usage choices this config supports, valid elemtents are "grid", "pv", "battery", "charge"
	Usages    []string
	Baudrate  int    // device specific default for modbus RS485 baudrate
	Comset    string // device specific default for modbus RS485 comset
}

type ParamBase struct {
	Params []Param
	Render string
}

var paramBaseList map[string]ParamBase

// Template describes is a proxy device for use with cli and automated testing
type Template struct {
	Template     string
	Description  string // user friendly description of the device this template describes
	Requirements Requirements
	GuidedSetup  GuidedSetup
	Generic      bool   // if this describes a generic device type rather than a product
	ParamsBase   string // references a base param set to inherit from
	Params       []Param
	Render       string // rendering template
}

// add the referenced base Params and overwrite existing ones
func (t *Template) ResolveParamBase() {
	if t.ParamsBase == "" {
		return
	}

	if paramBaseList == nil {
		err := yaml.Unmarshal([]byte(definition.ParamBaseListDefinition), &paramBaseList)
		if err != nil {
			fmt.Printf("Error: failed to parse paramBasesDefinition: %v\n", err)
			return
		}
	}

	base, ok := paramBaseList[t.ParamsBase]
	if !ok {
		fmt.Printf("Error: Could not find parambase definition: %s\n", t.ParamsBase)
		return
	}

	currentParams := make([]Param, len(t.Params))
	copy(currentParams, t.Params)
	t.Params = make([]Param, len(base.Params))
	copy(t.Params, base.Params)
	for _, p := range currentParams {
		if i, item := t.paramWithName(p.Name); item != nil {
			// we only allow overwriting a few fields
			if p.Default != "" {
				t.Params[i].Default = p.Default
			}
			if p.Example != "" {
				t.Params[i].Example = p.Example
			}
		} else {
			t.Params = append(t.Params, p)
		}
	}
}

// Defaults returns a map of default values for the template
func (t *Template) Defaults(docsOrTests bool) map[string]interface{} {
	values := make(map[string]interface{})
	for _, p := range t.Params {
		switch p.ValueType {
		case ParamValueTypeStringList:
			values[p.Name] = []string{}
		default:
			if p.Test != "" {
				values[p.Name] = p.Test
			} else if p.Example != "" && docsOrTests {
				values[p.Name] = p.Example
			} else {
				values[p.Name] = p.Default // may be empty
			}
		}
	}

	return values
}

// return the param with the given name
func (t *Template) paramWithName(name string) (int, *Param) {
	for i, p := range t.Params {
		if p.Name == name {
			return i, &p
		}
	}
	return 0, nil
}

// Usages returns the list of supported usages
func (t *Template) Usages() []string {
	if _, p := t.paramWithName(ParamUsage); p != nil {
		return p.Choice
	}

	return nil
}

func (t *Template) ModbusChoices() []string {
	if _, p := t.paramWithName(ParamModbus); p != nil {
		return p.Choice
	}

	return nil
}

//go:embed proxy.tpl
var proxyTmpl string

// RenderProxy renders the proxy template for inclusion in documentation
func (t *Template) RenderProxy() ([]byte, error) {
	return t.RenderProxyWithValues(nil, false)
}

func (t *Template) RenderProxyWithValues(values map[string]interface{}, includeDescription bool) ([]byte, error) {
	tmpl, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(proxyTmpl)
	if err != nil {
		panic(err)
	}

	for index, p := range t.Params {
		for k, v := range values {
			if p.Name != k {
				continue
			}

			switch p.ValueType {
			case ParamValueTypeStringList:
				for _, e := range v.([]string) {
					t.Params[index].Values = append(p.Values, yamlQuote(e))
				}
			default:
				t.Params[index].Value = yamlQuote(v.(string))
			}
		}
	}

	// remove params with no values
	var newParams []Param
	for _, param := range t.Params {
		if !param.Required {
			switch param.ValueType {
			case ParamValueTypeStringList:
				if len(param.Values) == 0 {
					continue
				}
			default:
				if param.Value == "" {
					continue
				}
			}
		}
		newParams = append(newParams, param)
	}

	t.Params = newParams

	out := new(bytes.Buffer)
	data := map[string]interface{}{
		"Template": t.Template,
		"Params":   t.Params,
	}
	if includeDescription {
		data["Description"] = t.Description
	}
	err = tmpl.Execute(out, data)

	return bytes.TrimSpace(out.Bytes()), err
}

// RenderResult renders the result template to instantiate the proxy
func (t *Template) RenderResult(docs bool, other map[string]interface{}) ([]byte, map[string]interface{}, error) {
	values := t.Defaults(docs)
	if err := util.DecodeOther(other, &values); err != nil {
		return nil, values, err
	}

	t.ModbusValues(values)

	// add the common templates
	for _, v := range paramBaseList {
		if !strings.Contains(t.Render, v.Render) {
			t.Render = fmt.Sprintf("%s\n%s", t.Render, v.Render)
		}
	}

	for item, p := range values {
		switch p := p.(type) {
		case []string:
			var list []string
			for _, v := range p {
				list = append(list, yamlQuote(v))
			}
			values[item] = list
		default:
			values[item] = yamlQuote(fmt.Sprintf("%v", p))
		}
	}

	tmpl := template.New("yaml")
	var funcMap template.FuncMap = map[string]interface{}{}
	// copied from: https://github.com/helm/helm/blob/8648ccf5d35d682dcd5f7a9c2082f0aaf071e817/pkg/engine/engine.go#L147-L154
	funcMap["include"] = func(name string, data interface{}) (string, error) {
		buf := bytes.NewBuffer(nil)
		if err := tmpl.ExecuteTemplate(buf, name, data); err != nil {
			return "", err
		}
		return buf.String(), nil
	}

	tmpl, err := tmpl.Funcs(template.FuncMap(sprig.FuncMap())).Funcs(funcMap).Parse(t.Render)
	if err != nil {
		return nil, values, err
	}

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, values)

	return bytes.TrimSpace(out.Bytes()), values, err
}
