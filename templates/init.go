package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sync"

	"gopkg.in/yaml.v3"
)

var (
	//go:embed charger/*.yaml meter/*.yaml vehicle/*.yaml
	yamlTemplates embed.FS

	templates = make(map[string][]Template)

	templatesOnce sync.Once
)

const (
	Charger = "charger"
	Meter   = "meter"
	Vehicle = "vehicle"
)

func loadTemplates(class string) {
	if templates[class] != nil {
		return
	}

	err := fs.WalkDir(yamlTemplates, ".", func(filepath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		b, err := fs.ReadFile(yamlTemplates, filepath)
		if err != nil {
			return err
		}

		var tmpl Template
		if err = yaml.Unmarshal(b, &tmpl); err != nil {
			panic(fmt.Errorf("reading template '%s' failed: %w", filepath, err))
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

func ByTemplate(t, class string) (Template, error) {
	loadTemplates(class)

	for _, tmpl := range templates[class] {
		if tmpl.Template == t {
			return tmpl, nil
		}
	}

	return Template{}, fmt.Errorf("template not found: %s", t)
}
