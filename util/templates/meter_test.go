package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMeterTemplateInjectsReversedParam(t *testing.T) {
	t.Parallel()

	for _, name := range []string{"eastron-sdm72", "homewizard-kwh"} {
		tmpl, err := ByName(Meter, name)
		require.NoError(t, err)

		_, param := tmpl.ParamByName(ParamReversed)
		require.Equal(t, ParamReversed, param.Name)
		require.Equal(t, TypeBool, param.Type)
		require.Equal(t, "false", param.Default)
	}
}

func TestMeterTemplateDoesNotRenderReversedFlag(t *testing.T) {
	t.Parallel()

	tmpl, err := ByName(Meter, "demo-meter")
	require.NoError(t, err)

	res, _, err := tmpl.RenderResult(RenderModeInstance, map[string]any{
		"template": "demo-meter",
		"usage":    "grid",
		"power":    123,
		"reversed": true,
	})
	require.NoError(t, err)
	require.NotContains(t, string(res), "reversed:")
}
