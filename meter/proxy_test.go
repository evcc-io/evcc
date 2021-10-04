package meter

import (
	"testing"

	"github.com/evcc-io/evcc/templates"
)

func TestProxies(t *testing.T) {
	for _, tmpl := range templates.ByClass(templates.Meter) {
		usages := tmpl.Usages()

		if len(usages) == 0 {
			t.Run(tmpl.Type, func(t *testing.T) {
				b, err := tmpl.RenderResult(nil)
				if err != nil {
					t.Log(string(b))
					t.Error(err)
				}
			})
		}

		// render all usages
		for _, usage := range usages {
			t.Run(tmpl.Type+" "+usage, func(t *testing.T) {
				b, err := tmpl.RenderResult(map[string]interface{}{
					"usage": usage,
				})

				if err != nil {
					t.Log(string(b))
					t.Error(err)
				}
			})
		}
	}
}
