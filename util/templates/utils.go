package templates

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"go.yaml.in/yaml/v4"
)

func yamlQuote(value string) string {
	if value == "" {
		return value
	}

	input := fmt.Sprintf("key: %s", value)

	var res struct {
		Value any `yaml:"key"`
	}

	if err := yaml.Unmarshal([]byte(input), &res); err == nil {
		b, err := yaml.Marshal(res)
		if err == nil && strings.TrimSpace(strings.TrimPrefix(string(b), "key: ")) == value {
			return value
		}
	}

	return quote(value)
}

func quote(value string) string {
	quoted := strings.ReplaceAll(value, `'`, `''`)
	return fmt.Sprintf("'%s'", quoted)
}

func trimLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, "\r\t ")
	}
	return strings.Join(lines, "\n")
}

func unquote(s string) string {
	return strings.Trim(s, `"'`)
}

// FuncMap returns a sprig template.FuncMap with additional include function
func FuncMap(tmpl *template.Template) *template.Template {
	funcMap := template.FuncMap{
		// include function
		// copied from: https://github.com/helm/helm/blob/8648ccf5d35d682dcd5f7a9c2082f0aaf071e817/pkg/engine/engine.go#L147-L154
		"include": func(name string, data interface{}) (string, error) {
			buf := bytes.NewBuffer(nil)
			if err := tmpl.ExecuteTemplate(buf, name, data); err != nil {
				return "", err
			}
			return buf.String(), nil
		},
		"urlEncode": url.QueryEscape,
		"unquote":   unquote,
		"quote":     yamlQuote,
	}

	return tmpl.Funcs(sprig.FuncMap()).Funcs(funcMap)
}
