package templates

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

func yamlQuote(value string) string {
	// don't quote empty strings
	if value == "" {
		return value
	}

	input := fmt.Sprintf("key: %s", value)

	var res struct {
		Value any `yaml:"key"`
	}

	if err := yaml.Unmarshal([]byte(input), &res); err != nil || value != res.Value {
		quoted := strings.ReplaceAll(value, `'`, `''`)
		return fmt.Sprintf("'%s'", quoted)
	}

	return value
}
