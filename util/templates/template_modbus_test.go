package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var renderModeNames = map[int]string{
	RenderModeInstance: "instance",
	RenderModeDocs:     "docs",
	RenderModeUnitTest: "unittest",
}

// TestModbusTemplateDefaultID verifies that a template-specific modbus id
// (e.g. Wallbe/Phoenix controllers using id 255) is rendered into the resulting
// instance config when the user did not supply an explicit id. See #29804.
func TestModbusTemplateDefaultID(t *testing.T) {
	for _, mode := range []int{RenderModeInstance, RenderModeDocs, RenderModeUnitTest} {
		t.Run(renderModeNames[mode], func(t *testing.T) {
			tmpl, err := ByName(Charger, "phoenix-ev-eth")
			require.NoError(t, err)

			_, values, err := tmpl.RenderResult(mode, map[string]any{
				"host": "192.168.0.8",
				"port": 502,
			})
			require.NoError(t, err)
			assert.Equal(t, "255", values["id"], "template-specific modbus id must be applied")
		})
	}
}

// TestModbusTemplateUserIDOverridesTemplate ensures a user-supplied id wins
// over the template default in all render modes.
func TestModbusTemplateUserIDOverridesTemplate(t *testing.T) {
	for _, mode := range []int{RenderModeInstance, RenderModeDocs, RenderModeUnitTest} {
		t.Run(renderModeNames[mode], func(t *testing.T) {
			tmpl, err := ByName(Charger, "phoenix-ev-eth")
			require.NoError(t, err)

			_, values, err := tmpl.RenderResult(mode, map[string]any{
				"host": "192.168.0.8",
				"port": 502,
				"id":   42,
			})
			require.NoError(t, err)
			assert.Equal(t, "42", values["id"], "user-supplied modbus id must not be overwritten")
		})
	}
}

// TestModbusParamHelpOverride verifies that a template may declare a modbus
// param to override its help text: it keeps that help, still inherits the
// definition's default, and does not suppress the remaining modbus params.
func TestModbusParamHelpOverride(t *testing.T) {
	ConfigDefaults.Load()

	tmpl := &Template{
		Params: []Param{
			{Name: ParamModbus, Choice: []string{ModbusChoiceTCPIP}},
			{Name: ModbusParamId, Help: TextLanguage{EN: "unit id of the meter"}},
		},
	}
	require.NoError(t, tmpl.UpdateModbusParamsWithDefaults())

	_, id := tmpl.ParamByName(ModbusParamId)
	assert.Equal(t, "unit id of the meter", id.Help.EN, "template help must survive")
	assert.Equal(t, "1", id.Default, "default must be inherited from the modbus definition")

	// the declared id must not stop host/port from being added
	values := map[string]any{ParamModbus: ModbusChoiceTCPIP}
	tmpl.ModbusParams("", values)

	for _, name := range []string{ModbusParamId, "host", ModbusParamPort} {
		i, _ := tmpl.ParamByName(name)
		assert.GreaterOrEqual(t, i, 0, "missing modbus param %s", name)
	}

	// adding is idempotent
	before := len(tmpl.Params)
	tmpl.ModbusParams("", values)
	assert.Equal(t, before, len(tmpl.Params), "repeated call must not duplicate params")
}

// TestWallbeTemplateCoveredByPhoenix verifies the BC migration: a config that
// still references the removed `wallbe` templates is transparently routed to
// the phoenix-ev-eth template via the `covers:` directive, while still
// producing the Wallbe controller's modbus slave id 255.
func TestWallbeTemplateCoveredByPhoenix(t *testing.T) {
	for _, name := range []string{"wallbe", "wallbe-meter", "wallbe-pre2019", "wallbe-pre2019-meter"} {
		t.Run(name, func(t *testing.T) {
			tmpl, err := ByName(Charger, name)
			require.NoError(t, err)
			assert.Equal(t, "phoenix-ev-eth", tmpl.Template)

			_, values, err := tmpl.RenderResult(RenderModeInstance, map[string]any{
				"host": "192.168.0.8",
				"port": 502,
			})
			require.NoError(t, err)
			assert.Equal(t, "255", values["id"])
		})
	}
}
