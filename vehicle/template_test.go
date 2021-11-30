package vehicle

import (
	"testing"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/util/test"
)

func TestVehicleTemplates(t *testing.T) {
	test.SkipCI(t)

	for _, tmpl := range templates.ByClass(templates.Vehicle) {
		tmpl := tmpl

		// set default values for all params
		values := tmpl.Defaults(true)

		// set the template value which is needed for rendering
		values["template"] = tmpl.Template

		t.Run(tmpl.Template, func(t *testing.T) {
			t.Parallel()

			b, values, err := tmpl.RenderResult(true, values)
			if err != nil {
				t.Logf("%s: %s", tmpl.Template, b)
				t.Error(err)
			}

			_, err = NewFromConfig("template", values)
			if err != nil && !test.Acceptable(err, acceptable) {
				t.Logf("%s", tmpl.Template)
				t.Error(err)
			}
		})
	}
}
