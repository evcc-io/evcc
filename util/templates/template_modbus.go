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

// add the modbus params to the template
func (t *Template) ModbusParams(values map[string]interface{}) {
	if len(t.ModbusChoices()) == 0 {
		return
	}

	if values[ParamModbus] == nil {
		return
	}

	for k := range values {
		switch k {
		case ModbusParamNameId:
			t.Params = append(t.Params, Param{Name: ModbusParamNameId, ValueType: ParamValueTypeNumber})
		case ModbusParamNameHost:
			if values[ParamModbus] != ModbusKeyTCPIP && values[ParamModbus] != ModbusKeyRS485TCPIP {
				continue
			}
			t.Params = append(t.Params, Param{Name: ModbusParamNameHost, ValueType: ParamValueTypeString})
		case ModbusParamNamePort:
			if values[ParamModbus] != ModbusKeyTCPIP && values[ParamModbus] != ModbusKeyRS485TCPIP {
				continue
			}
			t.Params = append(t.Params, Param{Name: ModbusParamNamePort, ValueType: ParamValueTypeNumber})
		case ModbusParamNameDevice:
			if values[ParamModbus] != ModbusKeyRS485Serial {
				continue
			}
			t.Params = append(t.Params, Param{Name: ModbusParamNameDevice, ValueType: ParamValueTypeString})
		case ModbusParamNameBaudrate:
			if values[ParamModbus] != ModbusKeyRS485Serial {
				continue
			}
			t.Params = append(t.Params, Param{Name: ModbusParamNameBaudrate, ValueType: ParamValueTypeNumber})
		case ModbusParamNameComset:
			if values[ParamModbus] != ModbusKeyRS485Serial {
				continue
			}
			t.Params = append(t.Params, Param{Name: ModbusParamNameComset, ValueType: ParamValueTypeString})
		}
	}

	delete(values, ParamModbus)
}

// set the modbus values required from modbus.tpl and and the template to the render
func (t *Template) ModbusValues(values map[string]interface{}) {
	if len(t.ModbusChoices()) == 0 {
		return
	}

	// either modbus param is defined or defaults for all modbus choices need to be set
	hasModbusValues := false
	if values[ModbusRS485Serial] == nil && values[ModbusRS485TCPIP] == nil && values[ModbusTCPIP] == nil {
		for k, v := range values {
			if k != ParamModbus {
				continue
			}

			switch s := v.(string); s {
			case ModbusKeyRS485Serial, ModbusKeyRS485TCPIP, ModbusKeyTCPIP:
				hasModbusValues = true
				values[s] = true
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
		return
	}

	// modbus defaults
	for k, v := range map[string]interface{}{
		ModbusParamNameId:       ModbusParamValueId,
		ModbusParamNameHost:     ModbusParamValueHost,
		ModbusParamNamePort:     ModbusParamValuePort,
		ModbusParamNameDevice:   ModbusParamValueDevice,
		ModbusParamNameBaudrate: ModbusParamValueBaudrate,
		ModbusParamNameComset:   ModbusParamValueComset,
	} {
		values[k] = v
	}

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
}
