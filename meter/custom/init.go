package custom

import (
	"embed"
	_ "embed"
	"fmt"
	"io/fs"

	"gopkg.in/yaml.v3"
)

//go:embed *.yaml
var yamlTemplates embed.FS

var templates []Template

func init() {
	var tmpl Template
	_ = fs.WalkDir(yamlTemplates, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		b, err := fs.ReadFile(yamlTemplates, path)
		if err != nil {
			return err
		}
		if err = yaml.Unmarshal(b, &tmpl); err != nil {
			panic(fmt.Errorf("reading template '%s' failed: %w", path, err))
		}
		templates = append(templates, tmpl)

		return nil
	})
}
