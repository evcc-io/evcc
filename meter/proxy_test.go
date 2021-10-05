package meter

import (
	"testing"

	"github.com/evcc-io/evcc/templates"
	"github.com/evcc-io/evcc/util/test"
)

func TestProxies(t *testing.T) {
	acceptable := []string{
		// `parsing "<no value>": invalid syntax`,
	}

	for _, tmpl := range templates.ByClass(templates.Meter) {
		usages := tmpl.Usages()

		if len(usages) == 0 {
			t.Run(tmpl.Type, func(t *testing.T) {
				b, err := tmpl.RenderResult(nil)
				if err != nil {
					t.Log(string(b))
					t.Error(err)
				}

				if _, err := NewFromConfig(tmpl.Type, nil); err != nil && !test.Acceptable(err, acceptable) {
					t.Error(err)
				}
			})
		}

		// render all usages
		for _, usage := range usages {
			usage := usage

			t.Run(tmpl.Type+" "+usage, func(t *testing.T) {
				t.Parallel()

				b, err := tmpl.RenderResult(map[string]interface{}{
					"usage": usage,
				})

				if err != nil {
					t.Log(string(b))
					t.Error(err)
				}

				if _, err := NewFromConfig(tmpl.Type, nil); err != nil && !test.Acceptable(err, acceptable) {
					t.Error(err)
				}
			})
		}
	}
}
