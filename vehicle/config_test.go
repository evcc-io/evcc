package vehicle

import (
	"testing"

	"github.com/andig/evcc/util/test"
)

func TestVehicles(t *testing.T) {
	acceptable := []string{
		"invalid plugin type: ...",
		"received status code 404 (INVALID PARAMS)",
		"missing personID",
		"401 Unauthorized",
		"unexpected length",
		"i/o timeout",
		"missing user",
	}

	for _, tmpl := range test.ConfigTemplates("vehicle") {
		_, err := NewFromConfig(tmpl.Type, tmpl.Config)
		if err != nil && !test.Acceptable(err, acceptable) {
			t.Logf("%s", tmpl.Name)
			t.Error(err)
		}
	}
}
