package util

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestYamlError(t *testing.T) {
	b := `
block:
  data: foo
  - mapped
`
	var res map[string]any
	err := yaml.Unmarshal([]byte(b), &res)
	require.Error(t, err)
	require.Equal(t, 2, yamlErrorLine(err))
}
