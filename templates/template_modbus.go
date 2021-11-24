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
	if values[ModbusRS485Serial] == nil && values[ModbusRS485TCPIP] == nil && values[ModbusTCPIP] == nil {
		for k, v := range values {
			if k != ParamModbus {
				continue
			}

			switch v.(string) {
			case ModbusKeyRS485Serial:
				hasModbusValues = true
				values[ModbusRS485Serial] = true
			case ModbusKeyRS485TCPIP:
				hasModbusValues = true
				values[ModbusRS485TCPIP] = true
			case ModbusKeyTCPIP:
				hasModbusValues = true
				values[ModbusTCPIP] = true
			default:
				// this happens during tests
				break
			}
			break
		}
	}

	// modbus defaults
	if !hasModbusValues {
		values[ModbusParamNameId] = ModbusParamValueId
		values[ModbusParamNameHost] = ModbusParamValueHost
		values[ModbusParamNamePort] = ModbusParamValuePort
		values[ModbusParamNameDevice] = ModbusParamValueDevice
		values[ModbusParamNameBaudrate] = ModbusParamValueBaudrate
		values[ModbusParamNameComset] = ModbusParamValueComset
		for _, p := range t.Params {
			if p.Name != ParamModbus {
				continue
			}
			for _, choice := range p.Choice {
				if !funk.ContainsString([]string{ModbusChoiceRS485, ModbusChoiceTCPIP}, choice) {
					return "", errors.New("Invalid modbus choice: " + choice)
				}
			}

			if funk.ContainsString(p.Choice, ModbusChoiceRS485) {
				values[ModbusRS485Serial] = true
				values[ModbusRS485TCPIP] = true
			}
			if funk.ContainsString(p.Choice, ModbusChoiceTCPIP) {
				values[ModbusTCPIP] = true
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
