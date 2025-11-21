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
