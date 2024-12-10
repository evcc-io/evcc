package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveParams(t *testing.T) {
	presetName := "preset1"

	ConfigDefaults.Presets[presetName] = preset{
		Params: []Param{
			{
				Name: "preset",
				Description: TextLanguage{
					Generic: "replacement description",
				},
			},
		},
	}

	tmpl := Template{
		TemplateDefinition: TemplateDefinition{
			Params: []Param{
				{
					Name: "simple",
					Description: TextLanguage{
						Generic: "simple description",
					},
				},
				{
					Name:   "preset",
					Preset: presetName,
					Description: TextLanguage{
						Generic: "preset1 description",
					},
				},
			},
		},
	}

	require.NoError(t, tmpl.ResolvePresets(Vehicle))

	_, p := tmpl.ParamByName("preset")
	require.Equal(t, "replacement description", p.Description.Generic)
}
