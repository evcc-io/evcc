package meter

import (
	"testing"

	"github.com/evcc-io/evcc/templates"
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
	"can only have either uri or device",               // modbus
	"(Client.Timeout exceeded while awaiting headers)", // http
}

func TestConfigMeters(t *testing.T) {
	test.SkipCI(t)

	for _, tmpl := range test.ConfigTemplates("meter") {
		tmpl := tmpl

		t.Run(tmpl.Name, func(t *testing.T) {
			t.Parallel()

			_, err := NewFromConfig(tmpl.Type, tmpl.Config)
			if err != nil && !test.Acceptable(err, acceptable) {
				t.Logf("%s: %+v", tmpl.Name, tmpl.Config)
				t.Error(err)
			}
		})
	}
}

func TestProxyMeters(t *testing.T) {
	test.SkipCI(t)

	for _, tmpl := range templates.ByClass(templates.Meter) {
		tmpl := tmpl

		values := tmpl.Defaults(true)

		// Modbus default test values
		if values["modbus"] != nil {
			modbusChoices := tmpl.ModbusChoices()
			if funk.ContainsString(modbusChoices, "tcpip") {
				values["modbustcpip"] = true
			} else {
				values["modbusrs485tcpip"] = true
			}
		}

		usages := tmpl.Usages()

		if len(usages) == 0 {
			runTest(t, tmpl, values)
		}

		// test all usages
		for _, usage := range usages {
			if usage != "" {
				values["usage"] = usage
			}

			runTest(t, tmpl, values)
		}
	}
}

func runTest(t *testing.T, tmpl templates.Template, values map[string]interface{}) {
	t.Run(tmpl.Type, func(t *testing.T) {
		t.Parallel()

		b, err := tmpl.RenderResult(false, values)
		if err != nil {
			t.Logf("%s: %s", tmpl.Type, b)
			t.Error(err)
		}

		if _, err := NewFromConfig(tmpl.Type, values); err != nil && !test.Acceptable(err, acceptable) {
			t.Logf("%s", tmpl.Type)
			t.Error(err)
		}
	})
}
