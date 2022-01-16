package templates

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed modbus.tpl
var modbusTmpl string

// add the modbus params to the template
func (t *Template) ModbusParams(modbusType string, values map[string]interface{}) {
	if len(t.ModbusChoices()) == 0 {
		return
	}

	if modbusType != "" {
		values[ParamModbus] = modbusType
	}

	if values[ParamModbus] == nil || values[ParamModbus] == "" {
		return
	}

	modbusParams := t.ConfigDefaults.Config.Modbus.Types[values[ParamModbus].(string)].Params
	// add the modbus params at the beginning
	t.Params = append(modbusParams, t.Params...)
}

// set the modbus values required from modbus.tpl and and the template to the render
func (t *Template) ModbusValues(renderMode string, values map[string]interface{}) {
	choices := t.ModbusChoices()
	if len(choices) == 0 {
		return
	}

	// only add the template once, when testing multiple usages, it might already be present
	if !strings.Contains(t.Render, modbusTmpl) {
		t.Render = fmt.Sprintf("%s\n%s", t.Render, modbusTmpl)
	}

	// either modbus param is defined, which means it ran through configuration
	// or defaults for all modbus choices need to be set for rendering all cases for documentation
	if modbusValue := values[ParamModbus]; renderMode != TemplateRenderModeInstance && modbusValue != nil && modbusValue != "" {
		values[fmt.Sprintf("%s", modbusValue)] = true
		return
	}

	modbusConfig := t.ConfigDefaults.Config.Modbus
	_, modbusParam := t.ParamByName(ParamModbus)

	modbusInterfaces := []string{}
	for _, choice := range choices {
		modbusInterfaces = append(modbusInterfaces, modbusConfig.Interfaces[choice]...)
	}

	for _, iface := range modbusInterfaces {
		typeParams := modbusConfig.Types[iface].Params
		for _, p := range typeParams {
			values[p.Name] = p.DefaultValue(renderMode)

			var defaultValue interface{}

			switch p.Name {
			case ModbusParamNameId:
				if modbusParam.ID != 0 {
					defaultValue = modbusParam.ID
				}
			case ModbusParamNamePort:
				if modbusParam.Port != 0 {
					defaultValue = modbusParam.Port
				}
			case ModbusParamNameBaudrate:
				if modbusParam.Baudrate != 0 {
					defaultValue = modbusParam.Baudrate
				}
			case ModbusParamNameComset:
				if modbusParam.Comset != "" {
					defaultValue = modbusParam.Comset
				}
			}

			if defaultValue == nil {
				continue
			}

			if renderMode == TemplateRenderModeInstance {
				t.SetParamDefault(p.Name, fmt.Sprintf("%d", defaultValue))
			} else {
				values[p.Name] = defaultValue
			}

		}
		if renderMode == TemplateRenderModeDocs {
			values[iface] = true
		}
	}
}
