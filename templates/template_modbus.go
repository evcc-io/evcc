package templates

import (
	"bytes"
	_ "embed"
	"errors"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/thoas/go-funk"
)

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
	if values["modbusrs485serial"] == nil && values["modbusrs485tcpip"] == nil && values["modbustcpip"] == nil {
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
