package charger

import (
	"testing"

	"github.com/andig/evcc/util/test"
)

func TestChargers(t *testing.T) {
	acceptable := []string{
		"invalid plugin type: ...",
		"mqtt not configured",
		"invalid charger type: nrgkick-bluetooth",
		"NRGKick bluetooth is only supported on linux",
		"invalid pin:",
		"connect: no route to host",
		"connect: connection refused",
		"missing password",
		"missing pin",
	}

	for _, tmpl := range test.ConfigTemplates("charger") {
		_, err := NewFromConfig(tmpl.Type, tmpl.Config)
		if err != nil && !test.Acceptable(err, acceptable) {
			t.Logf("%s", tmpl.Name)
			t.Error(err)
		}
	}
}
