package templates

import (
	"context"
	"errors"
	"maps"
	"slices"
	"testing"

	"github.com/evcc-io/evcc/plugin/auth"
	"github.com/evcc-io/evcc/util"
	"go.yaml.in/yaml/v4"
)

// test renders and instantiates plus yaml-parses the template per usage
func test(t *testing.T, tmpl Template, values map[string]any, cb func(values map[string]any)) {
	t.Helper()

	b, _, err := tmpl.RenderResult(RenderModeInstance, values)
	if err != nil {
		t.Log(string(b))
		t.Error(err)
		return
	}

	var instance any
	if err := yaml.Unmarshal(b, &instance); err != nil {
		t.Log(string(b))
		t.Error(err)
		return
	}

	// don't execute if skip test is set
	if slices.Contains(tmpl.Requirements.EVCC, RequirementSkipTest) {
		return
	}

	cb(values)
}

func testAuth(other map[string]any) error {
	if len(other) == 0 {
		return nil
	}

	var cc struct {
		Type   string
		Params []string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return err
	}

	params := make(map[string]any)
	for _, p := range cc.Params {
		params[p] = "foo"
	}

	_, err := auth.NewFromConfig(context.TODO(), cc.Type, params)

	// ConfigError indicates invalid parameters in mapstructure decode
	if ce := new(util.ConfigError); errors.As(err, &ce) {
		return err
	}

	return nil
}

func TestClass(t *testing.T, class Class, instantiate func(t *testing.T, values map[string]any)) {
	t.Parallel()

	for _, tmpl := range ByClass(class, WithDeprecated()) {
		// set default values for all params
		values := tmpl.Defaults(RenderModeUnitTest)

		// set modbus default test values
		if values[ParamModbus] != nil {
			modbusChoices := tmpl.ModbusChoices()
			// we only test one modbus setup
			if slices.Contains(modbusChoices, ModbusChoiceTCPIP) {
				values[ModbusKeyTCPIP] = true
			} else if slices.Contains(modbusChoices, ModbusChoiceUDP) {
				values[ModbusKeyUDP] = true
			} else {
				values[ModbusKeyRS485TCPIP] = true
			}
			tmpl.ModbusValues(RenderModeUnitTest, values)
		}

		// set the template value which is needed for rendering
		values["template"] = tmpl.Template
		// https://github.com/evcc-io/evcc/pull/10272 - override example IP (192.0.2.2)
		values["host"] = "localhost"

		// test auth configuration
		if err := testAuth(tmpl.TemplateDefinition.Auth); err != nil {
			t.Error("authorization:", err)
		}

		usages := tmpl.Usages()
		if len(usages) == 0 {
			t.Run(tmpl.Template, func(t *testing.T) {
				t.Parallel()

				test(t, tmpl, values, func(values map[string]any) {
					instantiate(t, values)
				})
			})

			continue
		}

		for _, u := range usages {
			// create a copy of the map for parallel execution
			usageValues := maps.Clone(values)
			usageValues[ParamUsage] = u

			// subtest for each usage
			t.Run(tmpl.Template+"/"+u, func(t *testing.T) {
				t.Parallel()

				test(t, tmpl, usageValues, func(values map[string]any) {
					instantiate(t, values)
				})
			})
		}
	}
}
