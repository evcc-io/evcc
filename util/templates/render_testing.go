package templates

import (
	"os"
	"testing"

	"github.com/jinzhu/copier"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

// renderTest renders and instantiates plus yaml-parses the template per usage
func renderTest(t *testing.T, tmpl Template, values map[string]interface{}, cb func(values map[string]interface{})) {
	t.Run(tmpl.Template, func(t *testing.T) {
		t.Parallel()

		_, values, err := tmpl.RenderResult(TemplateRenderModeUnitTest, values)
		if err != nil {
			t.Log(tmpl.Render)
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

func TestClass(t *testing.T, class Class, instantiate func(t *testing.T, values map[string]interface{})) {
	for _, tmpl := range ByClass(class) {
		tmpl := tmpl

		t.Run(tmpl.Template, func(t *testing.T) {
			// set default values for all params
			values := tmpl.Defaults(TemplateRenderModeUnitTest)

			// set the template value which is needed for rendering
			values["template"] = tmpl.Template

			// set modbus default test values
			if values[ParamModbus] != nil {
				modbusChoices := tmpl.ModbusChoices()
				// we only test one modbus setup
				if slices.Contains(modbusChoices, ModbusChoiceTCPIP) {
					values[ModbusKeyTCPIP] = true
				} else {
					values[ModbusKeyRS485TCPIP] = true
				}
				tmpl.ModbusValues(TemplateRenderModeInstance, values)
			}

			renderTest(t, tmpl, values, func(values map[string]interface{}) {
				instantiate(t, values)
			})
		})
	}
}
