package templates

import (
	"bytes"
	_ "embed"
	"errors"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/evcc-io/evcc/util"
	"github.com/thoas/go-funk"
)

const (
	ParamUsage         = "usage"
	ParamModbus        = "modbus"
	ModbusMagicComment = "# ::modbus-setup::"
)

type Param struct {
	Name    string
	Default string
	Hint    string
	Choice  []string
	Usages  []string
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

//go:embed modbus.tpl
var modbusTmpl string

func (t Template) RenderModbusTemplate(values map[string]interface{}, indentlength int) (string, error) {
	// indent the template
	lines := strings.Split(modbusTmpl, "\n")
	for i, line := range lines {
		lines[i] = strings.Repeat(" ", indentlength) + line
	}
	indentedTemplate := strings.Join(lines, "\n")

	tmpl, err := template.New("yaml").Funcs(template.FuncMap(sprig.FuncMap())).Parse(indentedTemplate)
	if err != nil {
		panic(err)
	}

	// either modbus param is defined or defaults for all modbus choices need to be set
	hasModbusValues := false
	for k, v := range values {
		if k != ParamModbus {
			continue
		}

		hasModbusValues = true
		switch v.(string) {
		case "rs485serial":
			values["modbusrs485serial"] = true
		case "rs485tcpip":
			values["modbusrs485tcpip"] = true
		case "tcpip":
			values["modbustcpip"] = true
		default:
			return "", errors.New("Invalid modbus value: " + v.(string))
		}
		break
	}

	// modbus defaults
	if !hasModbusValues {
		values["id"] = 1
		values["host"] = "192.0.2.2"
		values["port"] = 502
		values["device"] = "/dev/ttyUSB0"
		values["baudrate"] = 9600
		values["comset"] = "8N1"
		for _, p := range t.Params {
			if p.Name != ParamModbus {
				continue
			}
			for _, choice := range p.Choice {
				if !funk.ContainsString([]string{"rs485", "tcpip"}, choice) {
					return "", errors.New("Invalid modbus choice: " + choice)
				}
			}

			if funk.ContainsString(p.Choice, "rs485") {
				values["modbusrs485serial"] = true
				values["modbusrs485tcpip"] = true
			}
			if funk.ContainsString(p.Choice, "tcpip") {
				values["modbustcpip"] = true
			}
		}
	}

	out := new(bytes.Buffer)
	err = tmpl.Execute(out, values)

	return out.String(), err
}

func (t *Template) RenderModbus(values map[string]interface{}) error {
	// search for ModbusMagicComment and replace it with the correct indentation
	r := regexp.MustCompile(`.*` + ModbusMagicComment + `.*`)
	matches := r.FindAllString(t.Render, -1)
	for _, match := range matches {
		indentation := strings.Repeat(" ", strings.Index(match, ModbusMagicComment))
		result, err := t.RenderModbusTemplate(values, len(indentation))
		if err != nil {
			return err
		}
		if result != "" {
			t.Render = strings.ReplaceAll(t.Render, match, result)
		}
	}

	return nil
}

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
