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
