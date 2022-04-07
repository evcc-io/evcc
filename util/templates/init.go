package templates

import (
	"fmt"
	"io/fs"
	"path"

	"github.com/evcc-io/evcc/templates/definition"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

var (
	templates      = make(map[string][]Template)
	configDefaults = ConfigDefaults{}
)

const (
	Charger = "charger"
	Meter   = "meter"
	Vehicle = "vehicle"
)

func loadTemplates(class string) {
	configDefaults.LoadDefaults()

	if templates[class] != nil {
		return
	}

	err := fs.WalkDir(definition.YamlTemplates, ".", func(filepath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		b, err := fs.ReadFile(definition.YamlTemplates, filepath)
		if err != nil {
			return err
		}

		var definition TemplateDefinition
		if err = yaml.Unmarshal(b, &definition); err != nil {
			return fmt.Errorf("reading template '%s' failed: %w", filepath, err)
		}

		tmpl := Template{
			TemplateDefinition: definition,
			ConfigDefaults:     configDefaults,
		}
		if err = tmpl.ResolvePresets(); err != nil {
			return err
		}
		if err = tmpl.ResolveGroup(); err != nil {
			return err
		}
		if err = tmpl.UpdateParamsWithDefaults(); err != nil {
			return err
		}
		if err = tmpl.Validate(); err != nil {
			return err
		}

		path := path.Dir(filepath)
		templates[path] = append(templates[path], tmpl)

		return nil
	})

	if err != nil {
		panic(err)
	}
}

func ByClass(class string) []Template {
	loadTemplates(class)

	return templates[class]
}

func ByName(name, class string) (Template, error) {
	loadTemplates(class)

	for _, tmpl := range templates[class] {
		if tmpl.Template == name || slices.Contains(tmpl.Covers, name) {
			return tmpl, nil
		}
	}

	return Template{}, fmt.Errorf("template not found: %s", name)
}
