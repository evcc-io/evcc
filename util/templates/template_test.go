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

func TestValidatePattern(t *testing.T) {
	tmpl := &Template{
		TemplateDefinition: TemplateDefinition{
			Params: []Param{{Name: "host", Pattern: Pattern{Regex: `^[^\\/\s]+(:[0-9]{1,5})?$`}}},
		},
	}

	tests := []struct {
		host  string
		valid bool
	}{
		{"192.168.1.100", true},
		{"192.168.1.100:8080", true},
		{"example.com", true},
		{"http://192.168.1.100", false},
		{"192.168.1.100/admin", false},
		{"192.168.1.100 ", false},
	}

	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			_, _, err := tmpl.RenderResult(RenderModeInstance, map[string]any{"host": tt.host})
			if tt.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "does not match required pattern")
			}
		})
	}
}
