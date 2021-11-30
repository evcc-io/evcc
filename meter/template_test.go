package meter

import (
	"testing"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
	"github.com/thoas/go-funk"
)

func TestMeterTemplates(t *testing.T) {
	test.SkipCI(t)

	for _, tmpl := range templates.ByClass(templates.Meter) {
		tmpl := tmpl

		// set default values for all params
		values := tmpl.Defaults(true)

		// set the template value which is needed for rendering
		values["template"] = tmpl.Template

		// set modbus default test values
		if values[templates.ParamModbus] != nil {
			modbusChoices := tmpl.ModbusChoices()
			if funk.ContainsString(modbusChoices, templates.ModbusChoiceTCPIP) {
				values[templates.ModbusKeyTCPIP] = true
			} else {
				values[templates.ModbusKeyRS485TCPIP] = true
			}
		}

		usages := tmpl.Usages()
		if len(usages) == 0 {
			runTest(t, tmpl, values)
		} else {
			// test all usages
			for _, usage := range usages {

				// set the usage param value
				if usage != "" {
					values[templates.ParamUsage] = usage
				}

				runTest(t, tmpl, values)
			}
		}
	}
}

func runTest(t *testing.T, tmpl templates.Template, values map[string]interface{}) {
	t.Run(tmpl.Template, func(t *testing.T) {
		t.Parallel()

		b, values, err := tmpl.RenderResult(true, values)
		if err != nil {
			t.Logf("%s: %s", tmpl.Template, b)
			t.Error(err)
		}

		if _, err := NewFromConfig("template", values); err != nil && !test.Acceptable(err, acceptable) {
			t.Logf("%s", tmpl.Template)
			t.Error(err)
		}
	})
}
