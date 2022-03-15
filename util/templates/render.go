package templates

import (
	"os"
	"testing"

	"github.com/jinzhu/copier"
	"gopkg.in/yaml.v3"
)

// RenderTest renders and instantiates plus yaml-parses the template per usage
func RenderTest(t *testing.T, tmpl Template, values map[string]interface{}, cb func(values map[string]interface{})) {
	t.Run(tmpl.Template, func(t *testing.T) {
		t.Parallel()

		b, values, err := tmpl.RenderResult(TemplateRenderModeUnitTest, values)
		if err != nil {
			t.Log(string(b))
			t.Error(err)
		}

		// instantiate all usage variants
		for _, u := range tmpl.Usages() {
			t.Run(u, func(t *testing.T) {
				// t.Parallel()

				// create a copy of the map for parallel execution
				usageValues := make(map[string]interface{}, len(values)+1)
				if err := copier.Copy(&usageValues, values); err != nil {
					panic(err)
				}
				usageValues[ParamUsage] = u

				b, _, err := tmpl.RenderResult(TemplateRenderModeInstance, usageValues)
				if err != nil {
					t.Errorf("usage: %s, result: %v", u, err)
				}

				var instance interface{}
				if err := yaml.Unmarshal(b, &instance); err != nil {
					t.Errorf("usage: %s, yaml: %v", u, err)
				}

				// actually run the instance if not on CI
				if os.Getenv("CI") == "" {
					cb(usageValues)
				}
			})
		}
	})
}
