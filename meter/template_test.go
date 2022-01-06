package meter

import (
	"testing"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
	"github.com/thoas/go-funk"
)

var acceptable = []string{
	"invalid plugin type: ...",
	"missing mqtt broker configuration",
	"mqtt not configured",
	"not a SunSpec device",
	"missing password", // Powerwall
	"connect: no route to host",
	"connect: connection refused",
	"i/o timeout",
	"connect: network is unreachable",
	"no ping response for 192.0.2.2", // SMA
	"network is unreachable",
	"[1ESY1161052714 1ESY1161229249 1EMH0008842285 1ESY1161978584 1EMH0004864048 1ESY1161979033 7ELS8135823805]", // Discovergy
	"can only have either uri or device",                       // modbus
	"(Client.Timeout exceeded while awaiting headers)",         // http
	"cannot create meter 'discovergy': unexpected status: 401", //Discovergy Proxy
}

func TestMeterTemplates(t *testing.T) {
	test.SkipCI(t)

	for _, tmpl := range templates.ByClass(templates.Meter) {
		tmpl := tmpl

		// set default values for all params
		values := tmpl.Defaults(templates.TemplateRenderModeUnitTest)

		// set the template value which is needed for rendering
		values["template"] = tmpl.Template

		// set modbus default test values
		if values[templates.ParamModbus] != nil {
			modbusChoices := tmpl.ModbusChoices()
			// we only test one modbus setup
			if funk.ContainsString(modbusChoices, templates.ModbusChoiceTCPIP) {
				values[templates.ModbusTCPIP] = true
			} else {
				values[templates.ModbusRS485TCPIP] = true
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
		// t.Parallel()

		b, values, err := tmpl.RenderResult(templates.TemplateRenderModeUnitTest, values)
		if err != nil {
			t.Logf("Template: %s", tmpl.Template)
			t.Logf("%s", string(b))
			t.Error(err)
		}

		if _, err := NewFromConfig("template", values); err != nil && !test.Acceptable(err, acceptable) {
			t.Logf("Template: %s", tmpl.Template)
			t.Logf("%s", string(b))
			t.Error(err)
		}
	})
}
