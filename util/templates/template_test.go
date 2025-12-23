package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPresets(t *testing.T) {
	ConfigDefaults.Presets = map[string][]Param{
		"preset": {
			{Name: "preset 1"},
			{Name: "preset 2"},
		},
	}

	tmpl := &Template{
		TemplateDefinition: TemplateDefinition{
			Params: []Param{
				{Name: "first"},
				{Preset: "preset"},
				{Name: "last"},
			},
		},
	}

	require.NoError(t, tmpl.ResolvePresets())
	require.Equal(t, []Param{
		{Name: "first"},
		{Name: "preset 1"},
		{Name: "preset 2"},
		{Name: "last"},
	}, tmpl.Params)
}

func TestRequired(t *testing.T) {
	tmpl := &Template{
		TemplateDefinition: TemplateDefinition{
			Params: []Param{
				{
					Name:     "param",
					Required: true,
				},
			},
		},
	}

	_, _, err := tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"Param": "foo",
	})
	require.NoError(t, err)

	_, _, err = tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"Param": "",
	})
	require.Error(t, err)

	_, _, err = tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"Param": nil,
	})
	require.Error(t, err)

	_, _, err = tmpl.RenderResult(RenderModeDocs, map[string]any{
		"Param": nil,
	})
	require.NoError(t, err)
}
