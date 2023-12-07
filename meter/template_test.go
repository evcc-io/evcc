package meter

import (
	"testing"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
)

var acceptable = []string{
	"invalid plugin source: ...",
	"missing mqtt broker configuration",
	"missing token",
	"mqtt not configured",
	"not a SunSpec device",
	"missing credentials", // sockets
	"power: timeout",      // sockets
	"missing password",    // Powerwall
	"connect: no route to host",
	"connect: connection refused",
	"connect: network is unreachable",
	"i/o timeout",
	"'sma': missing uri or serial", // SMA
	"[1ESY1161052714 1ESY1161229249 1EMH0008842285 1ESY1161978584 1EMH0004864048 1ESY1161979033 7ELS8135823805]", // Discovergy
	"can only have either uri or device",               // modbus
	"(Client.Timeout exceeded while awaiting headers)", // http
	"context deadline exceeded",                        // LG ESS
	"no ping response for 192.0.2.2",                   // SMA
	"no such network interface",                        // SMA
}

func TestTemplates(t *testing.T) {
	templates.TestClass(t, templates.Meter, func(t *testing.T, values map[string]any) {
		t.Helper()

		if _, err := NewFromConfig("template", values); err != nil && !test.Acceptable(err, acceptable) {
			t.Log(values)
			t.Error(err)
		}
	})
}
