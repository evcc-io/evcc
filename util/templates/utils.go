package templates

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func quote(value string) string {
	quoted := strings.ReplaceAll(value, `'`, `''`)
	return fmt.Sprintf("'%s'", quoted)
}

func yamlQuote(value string) string {
	input := fmt.Sprintf("key: %s", value)

	var res struct {
		Value string `yaml:"key"`
	}

	if err := yaml.Unmarshal([]byte(input), &res); err != nil || value != res.Value {
		return quote(value)
	}

	// fix 0815, but not 0
	if strings.HasPrefix(value, "0") && len(value) > 1 {
		return quote(value)
	}

	return value
}
