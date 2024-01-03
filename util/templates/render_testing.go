package templates

import (
	"maps"
	"os"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

// test renders and instantiates plus yaml-parses the template per usage
func test(t *testing.T, tmpl Template, values map[string]interface{}, cb func(values map[string]interface{})) {
	t.Helper()

	b, _, err := tmpl.RenderResult(TemplateRenderModeInstance, values)
	if err != nil {
		t.Log(string(b))
		t.Error(err)
		return
	}

	var instance interface{}
	if err := yaml.Unmarshal(b, &instance); err != nil {
		t.Log(string(b))
		t.Error(err)
		return
	}

	// actually run the instance if not on CI
	if os.Getenv("CI") == "" {
		cb(values)
	}
}

func TestClass(t *testing.T, class Class, instantiate func(t *testing.T, values map[string]interface{})) {
	t.Parallel()

	for _, tmpl := range ByClass(class) {
		tmpl := tmpl

		// set default values for all params
		values := tmpl.Defaults(TemplateRenderModeUnitTest)

		// set modbus default test values
		if values[ParamModbus] != nil {
			modbusChoices := tmpl.ModbusChoices()
			// we only test one modbus setup
			if slices.Contains(modbusChoices, ModbusChoiceTCPIP) {
				values[ModbusKeyTCPIP] = true
			} else {
				values[ModbusKeyRS485TCPIP] = true
			}
			tmpl.ModbusValues(TemplateRenderModeUnitTest, values)
		}

		// set the template value which is needed for rendering
		values["template"] = tmpl.Template
		// https://github.com/evcc-io/evcc/pull/10272 - override example IP (192.0.2.2)
		values["host"] = "localhost"

		usages := tmpl.Usages()
		if len(usages) == 0 {
			t.Run(tmpl.Template, func(t *testing.T) {
				t.Parallel()

				test(t, tmpl, values, func(values map[string]interface{}) {
					instantiate(t, values)
				})
			})

			return
		}

		for _, u := range usages {
			// create a copy of the map for parallel execution
			usageValues := maps.Clone(values)
			usageValues[ParamUsage] = u

			// subtest for each usage
			t.Run(tmpl.Template+"/"+u, func(t *testing.T) {
				t.Parallel()

				test(t, tmpl, usageValues, func(values map[string]interface{}) {
					instantiate(t, values)
				})
			})
		}
	}
}
