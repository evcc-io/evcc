package vehicle

import (
	"testing"

	"github.com/andig/evcc/util/test"
)

func TestVehicles(t *testing.T) {
	for _, tmpl := range test.ConfigTemplates("vehicle") {
		_, err := NewFromConfig(tmpl.Type, tmpl.Config)
		if err != nil && !test.Acceptable("vehicle", err) {
			t.Logf("%s", tmpl.Name)
			t.Error(err)
		}
	}
}
