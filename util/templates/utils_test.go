package templates

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestYamlDecode(t *testing.T) {
	for _, value := range []string{`value`, `!value`, `@value`, `"value"`, `"va"lue"`, `va'lue`, `@va'lue`, `0815`, `4711`, `#pwd`, ``} {
		t.Run(value, func(t *testing.T) {
			quoted := yamlQuote(value)
			input := fmt.Sprintf("key: %s", quoted)

			var res struct {
				Value string `yaml:"key"`
			}

			if err := yaml.Unmarshal([]byte(input), &res); err != nil {
				t.Fatalf("expected %s (quoted: %s), got error %v", value, quoted, err)
			}

			if value != res.Value {
				t.Fatalf("expected %s (quoted: %s), got %s", value, quoted, res.Value)
			}
		})
	}
}

func TestYamlDecodeLeadingZero(t *testing.T) {
	exp := "'0815'"

	if res := yamlQuote("0815"); res != exp {
		t.Fatalf("expected %s, got %s", exp, res)
	}
}
