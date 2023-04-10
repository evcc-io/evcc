package charger

import (
	"testing"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
)

var acceptable = []string{
	"invalid plugin source: ...",
	"missing mqtt broker configuration",
	"mqtt not configured",
	"invalid charger type: nrgkick-bluetooth",
	"NRGKick bluetooth is only supported on linux",
	"invalid pin:",
	"hciconfig provided no response",
	"connect: no route to host",
	"connect: connection refused",
	"error connecting: Network Error",
	"i/o timeout",
	"recv timeout",
	"(Client.Timeout exceeded while awaiting headers)",
	"can only have either uri or device", // modbus
	"sponsorship required, see https://github.com/evcc-io/evcc#sponsorship",
	"eebus not configured",
	"context deadline exceeded",
	"missing credentials",
	"timeout",                    // ocpp
	"must have uri and password", // Wattpilot
}

func TestTemplates(t *testing.T) {
	templates.TestClass(t, templates.Charger, func(t *testing.T, values map[string]any) {
		if _, err := NewFromConfig("template", values); err != nil && !test.Acceptable(err, acceptable) {
			t.Log(values)
			t.Error(err)
		}
	})
}
