package templates

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveParams(t *testing.T) {
	presetName := "vehicle_common"

	override := Param{
		Name: "override",
		Description: TextLanguage{
			Generic: "override description",
		},
	}

	ConfigDefaults.Presets[presetName] = preset{
		Params: []Param{override},
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
					Name: override.Name,
				},
				{
					Preset: presetName,
				},
			},
		},
	}

	require.NoError(t, tmpl.ResolvePresets(Vehicle))

	_, p := tmpl.ParamByName("override")
	require.Equal(t, override.Description.Generic, p.Description.Generic)
}
