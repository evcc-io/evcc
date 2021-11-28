package templates

import (
	_ "embed"
	"errors"
	"fmt"
	"strings"

	"github.com/thoas/go-funk"
)

//go:embed modbus.tpl
var modbusTmpl string

// set the modbus values required from modbus.tpl and and the template to the render
func (t *Template) ModbusValues(values map[string]interface{}) map[string]interface{} {
	if len(t.ModbusChoices()) == 0 {
		return values
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
			}
			break
		}
	}

	// only add the template once, when testing multiple usages, it might already be present
	if !strings.Contains(t.Render, modbusTmpl) {
		t.Render = fmt.Sprintf("%s\n%s", t.Render, modbusTmpl)
	}

	if hasModbusValues {
		return values
	}

	// modbus defaults
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
				panic(errors.New("Invalid modbus choice: " + choice))
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

	return values
}
