package meter

import (
	"testing"

	"github.com/evcc-io/evcc/templates"
)

func TestProxies(t *testing.T) {
	for _, tmpl := range templates.ByClass(templates.Meter) {
		usages := tmpl.Usages()
		if len(usages) == 0 {
			t.Log(tmpl.Type + " - all")

			b, err := tmpl.RenderResult(nil)
			if err != nil {
				t.Log(string(b))
				t.Error(err)
			}
		}

		// render all usages
		for _, usage := range usages {
			t.Log(tmpl.Type + " - " + usage)

			b, err := tmpl.RenderResult(map[string]interface{}{
				"usage": usage,
			})

			if err != nil {
				t.Log(string(b))
				t.Error(err)
			}
		}
	}
}
