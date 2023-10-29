package templates

import (
	"bytes"
	"fmt"
	"net/url"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"gopkg.in/yaml.v3"
)

func quote(value string) string {
	quoted := strings.ReplaceAll(value, `'`, `''`)
	return fmt.Sprintf("'%s'", quoted)
}

// yamlQuote quotes strings for yaml if they would otherwise by modified by the unmarshaler
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

func trimLines(s string) string {
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lines[i] = strings.TrimRight(line, "\r\t ")
	}
	return strings.Join(lines, "\n")
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
		"urlEncode": func(v string) string {
			return url.QueryEscape(v)
		},
	}

	return tmpl.Funcs(sprig.TxtFuncMap()).Funcs(funcMap)
}
