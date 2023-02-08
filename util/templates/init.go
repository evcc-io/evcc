package templates

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"

	"github.com/evcc-io/evcc/templates/definition"
	"golang.org/x/exp/slices"
	"gopkg.in/yaml.v3"
)

var (
	templates      = make(map[Class][]Template)
	configDefaults = ConfigDefaults{}
)

func init() {
	configDefaults.LoadDefaults()

	loadTemplates(Charger)
	loadTemplates(Meter)
	loadTemplates(Vehicle)
}

func FromBytes(b []byte) (Template, error) {
	// panic if template definition contains unknown fields
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)

	var definition TemplateDefinition
	if err := dec.Decode(&definition); err != nil {
		return Template{}, err
	}

	tmpl := Template{
		TemplateDefinition: definition,
		ConfigDefaults:     configDefaults,
	}

	err := tmpl.ResolvePresets()
	if err == nil {
		err = tmpl.ResolveGroup()
	}
	if err == nil {
		err = tmpl.UpdateParamsWithDefaults()
	}
	if err == nil {
		err = tmpl.Validate()
	}

	// set default value type
	for _, p := range tmpl.Params {
		if p.Type == "" {
			p.Type = ParamTypeString
		}
	}

	return tmpl, err
}

func loadTemplates(class Class) {
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

		tmpl, err := FromBytes(b)
		if err != nil {
			return fmt.Errorf("processing template '%s' failed: %w", filepath, err)
		}

		class, err := ClassString(path.Dir(filepath))
		if err != nil {
			return fmt.Errorf("invalid template class: '%s'", err)
		}

		templates[class] = append(templates[class], tmpl)

		return nil
	})

	if err != nil {
		panic(err)
	}
}

func ByClass(class Class) []Template {
	return templates[class]
}

func ByName(class Class, name string) (Template, error) {
	for _, tmpl := range templates[class] {
		if tmpl.Template == name || slices.Contains(tmpl.Covers, name) {
			return tmpl, nil
		}
	}

	return Template{}, fmt.Errorf("template not found: %s", name)
}
