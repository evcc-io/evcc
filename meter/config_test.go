package meter

import (
	"testing"

	"github.com/evcc-io/evcc/util/test"
)

func TestMeters(t *testing.T) {
	test.SkipCI(t)

	acceptable := []string{
		"invalid plugin type: ...",
		"missing mqtt broker configuration",
		"mqtt not configured",
		"not a SunSpec device",
		"missing password", // Powerwall
		"connect: no route to host",
		"connect: connection refused",
		"i/o timeout",
		"no ping response for 192.0.2.2", // SMA
		"[1ESY1161052714 1ESY1161229249 1EMH0008842285 1ESY1161978584 1EMH0004864048 1ESY1161979033 7ELS8135823805]", // Discovergy
		"can only have either uri or device", // modbus
	}

	for _, tmpl := range test.ConfigTemplates("meter") {
		t.Run(tmpl.Name, func(t *testing.T) {
			_, err := NewFromConfig(tmpl.Type, tmpl.Config)
			if err != nil && !test.Acceptable(err, acceptable) {
				t.Logf("%s: %+v", tmpl.Name, tmpl.Config)
				t.Error(err)
			}
		})
	}
}
