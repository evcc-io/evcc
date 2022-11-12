package meter

import (
	"testing"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
	"golang.org/x/exp/slices"
)

var acceptable = []string{
	"invalid plugin source: ...",
	"missing mqtt broker configuration",
	"mqtt not configured",
	"not a SunSpec device",
	"missing password",         // Powerwall
	"missing user or password", // Tapo
	"connect: no route to host",
	"connect: connection refused",
	"connect: network is unreachable",
	"i/o timeout",
	"'sma': missing uri or serial", // SMA
	"'fritzdect': missing ain",     // FritzDect
	"[1ESY1161052714 1ESY1161229249 1EMH0008842285 1ESY1161978584 1EMH0004864048 1ESY1161979033 7ELS8135823805]", // Discovergy
	"can only have either uri or device",                                          // modbus
	"(Client.Timeout exceeded while awaiting headers)",                            // http
	"unexpected status: 401",                                                      // Discovergy
	"unexpected status: 503",                                                      // Discovergy
	"login failed: Put \"https://192.0.2.2/v1/login\": context deadline exceeded", // LG ESS
}

func TestTemplates(t *testing.T) {
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
			if slices.Contains(modbusChoices, templates.ModbusChoiceTCPIP) {
				values[templates.ModbusKeyTCPIP] = true
			} else {
				values[templates.ModbusKeyRS485TCPIP] = true
			}
			tmpl.ModbusValues(templates.TemplateRenderModeInstance, values)
		}

		templates.RenderTest(t, tmpl, values, func(values map[string]interface{}) {
			if _, err := NewFromConfig("template", values); err != nil && !test.Acceptable(err, acceptable) {
				t.Log(values)
				t.Error(err)
			}
		})
	}
}
