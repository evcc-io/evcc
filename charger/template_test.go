package charger

import (
	"context"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
)

var acceptable = []string{
	api.ErrMissingCredentials.Error(),
	api.ErrMissingToken.Error(),
	"invalid plugin source: ...",
	"missing mqtt broker configuration",
	"mqtt not configured",
	"invalid charger type: nrgkick-bluetooth",
	"NRGKick bluetooth is only supported on linux",
	"invalid pin:",
	"hciconfig provided no response",
	"connect: no route to host",
	"connect: connection refused",
	"connector already registered: 1", // ocpp
	"error connecting: Network Error",
	"i/o timeout",
	"loadpoint 1 is not configured", // openWB
	"recv timeout",
	"(Client.Timeout exceeded while awaiting headers)",
	"can only have either uri or device",                                   // modbus
	"connection already registered with different protocol: localhost:502", // modbus
	"sponsorship required, see https://docs.evcc.io/docs/sponsorship",
	"eebus not configured",
	"context deadline exceeded",
	"timeout",                              // ocpp
	"must have uri and password",           // Wattpilot
	"either identity or uuid are required", // Plugchoice
}

func TestTemplates(t *testing.T) {
	templates.TestClass(t, templates.Charger, func(t *testing.T, values map[string]any) {
		t.Helper()
		if _, err := NewFromConfig(context.TODO(), "template", values); err != nil && !test.Acceptable(err, acceptable) {
			t.Log(values)
			t.Error(err)
		}
	})
}
