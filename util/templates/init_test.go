package templates

import (
	"testing"
	"testing/fstest"

	"github.com/stretchr/testify/require"
)

func TestClassLocalIncludes(t *testing.T) {
	fsys := fstest.MapFS{
		"loadpoint/local" + IncludeExt: &fstest.MapFile{
			Data: []byte(`{{ define "local" }}power: {{ .foo }}{{ end }}`),
		},
	}

	require.NoError(t, loadIncludes(fsys, Loadpoint))
	require.Contains(t, classTmpl, Loadpoint)

	// class without include files
	require.NoError(t, loadIncludes(fsys, Circuit))
	require.NotContains(t, classTmpl, Circuit)

	tmpl := Template{
		Params: []Param{{Name: "foo", Default: "bar"}},
		Render: "type: custom\n{{ include \"local\" . }}",
	}

	b, _, err := tmpl.RenderResult(Loadpoint, RenderModeInstance, map[string]any{})
	require.NoError(t, err)
	require.Equal(t, "type: custom\npower: bar", string(b))

	// class-local includes are not visible to other classes
	_, _, err = tmpl.RenderResult(Circuit, RenderModeInstance, map[string]any{})
	require.Error(t, err)
}
