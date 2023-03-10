package templates

import (
	"testing"

	"golang.org/x/exp/slices"
)

func TestClass(t *testing.T, class Class, instantiate func(t *testing.T, values map[string]interface{})) {
	for _, tmpl := range ByClass(class) {
		tmpl := tmpl

		t.Run(tmpl.Template, func(t *testing.T) {
			// set default values for all params
			values := tmpl.Defaults(TemplateRenderModeUnitTest)

			// set the template value which is needed for rendering
			values["template"] = tmpl.Template

			// set modbus default test values
			if values[ParamModbus] != nil {
				modbusChoices := tmpl.ModbusChoices()
				// we only test one modbus setup
				if slices.Contains(modbusChoices, ModbusChoiceTCPIP) {
					values[ModbusKeyTCPIP] = true
				} else {
					values[ModbusKeyRS485TCPIP] = true
				}
				tmpl.ModbusValues(TemplateRenderModeInstance, values)
			}

			RenderTest(t, tmpl, values, func(values map[string]interface{}) {
				instantiate(t, values)
			})
		})
	}
}
