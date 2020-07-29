package charger

import (
	"testing"

	"github.com/andig/evcc/util/test"
)

func TestChargers(t *testing.T) {
	for _, tmpl := range test.ConfigTemplates("charger") {
		_, err := NewFromConfig(tmpl.Type, tmpl.Config)
		if err != nil && !test.Acceptable("charger", err) {
			t.Logf("%s", tmpl.Name)
			t.Error(err)
		}
	}
}
