package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestYamlFloat(t *testing.T) {
	b := `example: 55.7351`
	var res map[string]string
	require.NoError(t, yaml.Unmarshal([]byte(b), &res))
}

func TestYamlEmpty(t *testing.T) {
	var res map[string]any
	err := yaml.Unmarshal([]byte(""), &res)
	require.ErrorContains(t, err, "no documents in stream")
}

func TestYamlCommentsOnly(t *testing.T) {
	b := `# just a comment
# another comment
`
	var res map[string]any
	err := yaml.Unmarshal([]byte(b), &res)
	require.ErrorContains(t, err, "no documents in stream")
}

func TestYamlError(t *testing.T) {
	b := `block:
  data: foo
  - mapped`

	var res map[string]any
	err := yaml.Unmarshal([]byte(b), &res)

	require.Error(t, err)
	require.Equal(t, 3, yamlErrorLine(err))
}
