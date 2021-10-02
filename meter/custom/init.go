package custom

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

//go:embed sunspec.yaml
var sunspec string

var templates []Template

func init() {
	var tmpl Template
	if err := yaml.Unmarshal([]byte(sunspec), &tmpl); err != nil {
		panic(fmt.Errorf("reading template failed: %w", err))
	}

	templates = append(templates, tmpl)
}
