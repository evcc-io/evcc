package meter

import (
	"testing"

	"github.com/andig/evcc/util/test"
)

func TestMeters(t *testing.T) {
	for _, tmpl := range test.ConfigTemplates("meter") {
		_, err := NewFromConfig(tmpl.Type, tmpl.Config)
		if err != nil && !test.Acceptable("meter", err) {
			t.Logf("%s", tmpl.Name)
			t.Error(err)
		}
	}
}
