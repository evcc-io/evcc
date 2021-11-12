package templates

import (
	"embed"
	"fmt"
	"io/fs"
	"path"

	"gopkg.in/yaml.v3"
)

var (
	//go:embed charger/*.yaml meter/*.yaml vehicle/*.yaml
	yamlTemplates embed.FS

	templates = make(map[string][]Template)
)

const (
	Charger = "charger"
	Meter   = "meter"
	Vehicle = "vehicle"
)

//go:generate go run generate/generate.go
func init() {
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
	return templates[class]
}

func ByType(t, class string) Template {
	for _, tmpl := range templates[class] {
		if tmpl.Type == t {
			return tmpl
		}
	}

	return Template{}
}
