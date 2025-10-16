package templates

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v4"
)

func TestYamlDecode(t *testing.T) {
	for _, value := range []string{`value`, `!value`, `@value`, `"va"lue"`, `va'lue`, `@va'lue`, `0815`, `4711`, `#pwd`, ``} {
		t.Run(value, func(t *testing.T) {
			quoted := yamlQuote(value)
			input := fmt.Sprintf("key: %s", quoted)

			var res struct {
				Value string `yaml:"key"`
			}

			err := yaml.Unmarshal([]byte(input), &res)
			require.NoError(t, err)
			assert.Equal(t, value, res.Value)
		})
	}
}

func TestYamlDecodeLeadingZero(t *testing.T) {
	p := Param{Type: TypeString}
	assert.Equal(t, "'0815'", p.yamlQuote("0815"))
}

func TestYamlQuote(t *testing.T) {
	assert.Equal(t, `"a\nb\nc"`, yamlQuote(`a
b
c`))
}
