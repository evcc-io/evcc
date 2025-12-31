package templates

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
	assert.NoError(t, err, "test: required present")

	_, _, err = tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"Param": "",
	})
	assert.Error(t, err, "test: required present but empty")

	_, _, err = tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"Param": nil,
	})
	assert.Error(t, err, "test: required present but nil")

	_, _, err = tmpl.RenderResult(RenderModeDocs, map[string]any{
		"Param": nil,
	})
	assert.NoError(t, err, "docs: required present but nil")
}

func TestRequiredDeprecated(t *testing.T) {
	tmpl := &Template{
		TemplateDefinition: TemplateDefinition{
			Params: []Param{
				{
					Name:       "param",
					Required:   true,
					Deprecated: true,
				},
			},
		},
	}

	_, _, err := tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"Param": "foo",
	})
	assert.NoError(t, err, "test: required present")

	_, _, err = tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"Param": "",
	})
	assert.NoError(t, err, "test: required present but empty")

	_, _, err = tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"Param": nil,
	})
	assert.NoError(t, err, "test: required present but nil")

	_, _, err = tmpl.RenderResult(RenderModeDocs, map[string]any{
		"Param": nil,
	})
	assert.NoError(t, err, "docs: required present but nil")
}

func TestRequiredPerUsage(t *testing.T) {
	tmpl := &Template{
		TemplateDefinition: TemplateDefinition{
			Params: []Param{
				{
					Name: "usage",
				},
				{
					Name:     "param",
					Required: true,
					Usages:   []string{"battery"},
				},
			},
		},
	}

	_, _, err := tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"Param": nil,
		"Usage": nil,
	})
	require.NoError(t, err)

	_, _, err = tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"Param": nil,
		"Usage": "pv",
	})
	require.NoError(t, err)

	_, _, err = tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"Param": nil,
		"Usage": "battery",
	})
	require.Error(t, err)

	_, _, err = tmpl.RenderResult(RenderModeUnitTest, map[string]any{
		"Param": "foo",
		"Usage": "battery",
	})
	require.NoError(t, err)
}
