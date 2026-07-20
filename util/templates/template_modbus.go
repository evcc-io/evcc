package templates

import (
	_ "embed"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

//go:embed modbus.tpl
var modbusTmpl string

// ModbusParams adds the modbus parameters' default values
func (t *Template) ModbusParams(modbusType string, values map[string]any) {
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

	modbusParams := ConfigDefaults.Modbus.Types[values[ParamModbus].(string)].Params

	// add the modbus params at the beginning
	t.Params = append(modbusParams, t.Params...)
}

// AddModbusCommonParams adds the connection-independent modbus params (delay,
// timeout) so they are available as regular advanced params. Device-specific
// defaults from the modbus param are applied to existing params, too.
func (t *Template) AddModbusCommonParams() error {
	if len(t.ModbusChoices()) == 0 {
		return nil
	}

	_, modbusParam := t.ParamByName(ParamModbus)

	for _, common := range []struct {
		name, def string
	}{
		{ModbusParamDelay, modbusParam.Delay},
		{ModbusParamTimeout, modbusParam.Timeout},
	} {
		name, def := common.name, common.def

		if i, _ := t.ParamByName(name); i >= 0 {
			// existing (typically deprecated) param: only apply the default
			if def != "" {
				t.SetParamDefault(name, def)
			}
			continue
		}

		if i := slices.IndexFunc(ConfigDefaults.Modbus.Definitions, func(p Param) bool {
			return p.Name == name
		}); i >= 0 {
			p := ConfigDefaults.Modbus.Definitions[i]
			if def != "" {
				p.Default = def
			}
			t.Params = append(t.Params, p)
		}
	}

	return nil
}

// ModbusValues adds the values required for modbus.tpl to the value map
func (t *Template) ModbusValues(renderMode int, values map[string]any) {
	choices := t.ModbusChoices()
	if len(choices) == 0 {
		return
	}

	// only add the template once, when testing multiple usages, it might already be present
	if !strings.Contains(t.Render, modbusTmpl) {
		t.Render = fmt.Sprintf("%s\n%s", t.Render, modbusTmpl)
	}

	modbusConfig := ConfigDefaults.Modbus
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
			case ModbusParamId:
				if modbusParam.ID != 0 {
					defaultValue = strconv.Itoa(modbusParam.ID)
				}
			case ModbusParamPort:
				if modbusParam.Port != 0 {
					defaultValue = strconv.Itoa(modbusParam.Port)
				}
			case ModbusParamBaudrate:
				if modbusParam.Baudrate != 0 {
					defaultValue = strconv.Itoa(modbusParam.Baudrate)
				}
			case ModbusParamComset:
				if modbusParam.Comset != "" {
					defaultValue = modbusParam.Comset
				}
			}

			if defaultValue != "" {
				// apply the template-specific default to both the render values
				// (so RenderModeInstance YAML reflects it) and the param definition
				// (so the Config UI surfaces it as default). The earlier guard above
				// ensures user-supplied values are not overwritten.
				values[p.Name] = defaultValue
				if renderMode == RenderModeInstance {
					t.SetParamDefault(p.Name, defaultValue)
				}
			}
		}

		if renderMode == RenderModeDocs {
			values[iface] = true
		}
	}
}
