package charger

import (
	"testing"

	"github.com/evcc-io/evcc/util/test"
)

var acceptable = []string{
	"invalid plugin type: ...",
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
}

func TestConfigChargers(t *testing.T) {
	test.SkipCI(t)

	for _, tmpl := range test.ConfigTemplates("charger") {
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
