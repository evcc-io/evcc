package charger

import (
	"testing"

	"github.com/mark-sch/evcc/util/test"
)

func TestChargers(t *testing.T) {
	acceptable := []string{
		"invalid plugin type: ...",
		"missing mqtt broker configuration",
		"mqtt not configured",
		"invalid charger type: nrgkick-bluetooth",
		"NRGKick bluetooth is only supported on linux",
		"invalid pin:",
		"connect: no route to host",
		"connect: connection refused",
		"error connecting: Network Error",
		"i/o timeout",
	}

	for _, tmpl := range test.ConfigTemplates("charger") {
		t.Run(tmpl.Name, func(t *testing.T) {
			_, err := NewFromConfig(tmpl.Type, tmpl.Config)
			if err != nil && !test.Acceptable(err, acceptable) {
				t.Logf("%s: %+v", tmpl.Name, tmpl.Config)
				t.Error(err)
			}
		})
	}
}
