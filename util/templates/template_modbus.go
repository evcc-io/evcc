package templates

import (
	_ "embed"
	"fmt"
	"strings"
)

//go:embed modbus.tpl
var modbusTmpl string

// ModbusParams adds the modbus parameters' default values
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

	// check if the modbus params are already added
	if index, _ := t.ParamByName("id"); index >= 0 {
		return
	}

	modbusParams := t.ConfigDefaults.Modbus.Types[values[ParamModbus].(string)].Params

	// add the modbus params at the beginning
	t.Params = append(modbusParams, t.Params...)
}

// ModbusValues adds the values required for modbus.tpl to the value map
func (t *Template) ModbusValues(renderMode string, values map[string]interface{}) {
	choices := t.ModbusChoices()
	if len(choices) == 0 {
		return
	}

	// only add the template once, when testing multiple usages, it might already be present
	if !strings.Contains(t.Render, modbusTmpl) {
		t.Render = fmt.Sprintf("%s\n%s", t.Render, modbusTmpl)
	}

	modbusConfig := t.ConfigDefaults.Modbus
	_, modbusParam := t.ParamByName(ParamModbus)

	modbusInterfaces := []string{}
	for _, choice := range choices {
		modbusInterfaces = append(modbusInterfaces, modbusConfig.Interfaces[choice]...)
	}

	// set default interface type
	if len(modbusInterfaces) == 1 {
		values[ParamModbus] = modbusInterfaces[0]
	}

	for _, iface := range modbusInterfaces {
		typeParams := modbusConfig.Types[iface].Params

		for _, p := range typeParams {
			// don't overwrite custom values
			if values[p.Name] != nil {
				continue
			}

			values[p.Name] = p.DefaultValue(renderMode)

			var defaultValue string

			switch p.Name {
			case ModbusParamNameId:
				if modbusParam.ID != 0 {
					defaultValue = fmt.Sprintf("%d", modbusParam.ID)
				}
			case ModbusParamNamePort:
				if modbusParam.Port != 0 {
					defaultValue = fmt.Sprintf("%d", modbusParam.Port)
				}
			case ModbusParamNameBaudrate:
				if modbusParam.Baudrate != 0 {
					defaultValue = fmt.Sprintf("%d", modbusParam.Baudrate)
				}
			case ModbusParamNameComset:
				if modbusParam.Comset != "" {
					defaultValue = modbusParam.Comset
				}
			}

			if defaultValue != "" {
				// for modbus params the default value is carried
				// using the parameter default, not the value
				// TODO figure out why that's necessary
				if renderMode == TemplateRenderModeInstance {
					t.SetParamDefault(p.Name, defaultValue)
				} else {
					values[p.Name] = defaultValue
				}
			}
		}

		if renderMode == TemplateRenderModeDocs {
			values[iface] = true
		}
	}
}
