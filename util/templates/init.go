package templates

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"path"
	"slices"
	"sync"
	"text/template"

	"github.com/evcc-io/evcc/templates/definition"
	"gopkg.in/yaml.v3"
)

var (
	//go:embed includes/*.tpl
	includeFS embed.FS

	// baseTmpl holds all included template definitions
	baseTmpl *template.Template

	templates       = make(map[Class][]Template)
	ConfigDefaults  configDefaults
	mu              sync.Mutex
	encoderLanguage string
)

func init() {
	ConfigDefaults.Load()

	baseTmpl = template.Must(template.ParseFS(includeFS, "includes/*.tpl"))

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

// EncoderLanguage sets the template language for encoding json
func EncoderLanguage(lang string) {
	mu.Lock()
	defer mu.Unlock()
	encoderLanguage = lang
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
