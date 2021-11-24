package vehicle

import (
	"testing"

	"github.com/evcc-io/evcc/templates"
	"github.com/evcc-io/evcc/util/test"
)

func TestProxyVehicles(t *testing.T) {
	test.SkipCI(t)

	for _, tmpl := range templates.ByClass(templates.Vehicle) {
		tmpl := tmpl

		values := tmpl.Defaults(true)

		t.Run(tmpl.Template, func(t *testing.T) {
			t.Parallel()

			b, err := tmpl.RenderResult(true, values)
			if err != nil {
				t.Logf("%s: %s", tmpl.Template, b)
				t.Error(err)
			}

			_, err = NewFromConfig("", tmpl.Template, values)
			if err != nil && !test.Acceptable(err, acceptable) {
				t.Logf("%s", tmpl.Template)
				t.Error(err)
			}
		})
	}
}
