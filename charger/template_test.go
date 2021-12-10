package charger

import (
	"testing"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
	"github.com/thoas/go-funk"
)

func TestChargerTemplates(t *testing.T) {
	test.SkipCI(t)

	for _, tmpl := range templates.ByClass(templates.Charger) {
		tmpl := tmpl

		// set default values for all params
		values := tmpl.Defaults(true)

		// set the template value which is needed for rendering
		values["template"] = tmpl.Template

		// set modbus default test values
		if values[templates.ParamModbus] != nil {
			modbusChoices := tmpl.ModbusChoices()
			if funk.ContainsString(modbusChoices, templates.ModbusChoiceTCPIP) {
				values[templates.ModbusTCPIP] = true
			} else {
				values[templates.ModbusRS485TCPIP] = true
			}
		}

		t.Run(tmpl.Template, func(t *testing.T) {
			t.Parallel()

			b, values, err := tmpl.RenderResult(true, values)
			if err != nil {
				t.Logf("%s: %s", tmpl.Template, b)
				t.Error(err)
			}

			_, err = NewFromConfig("template", values)
			if err != nil && !test.Acceptable(err, acceptable) {
				t.Logf("%s", tmpl.Template)
				t.Error(err)
			}
		})
	}
}
