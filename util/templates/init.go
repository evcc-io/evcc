package templates

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"slices"
	"sync"
	"text/template"

	"github.com/evcc-io/evcc/templates/definition"
	"github.com/samber/lo"
	"go.yaml.in/yaml/v4"
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

	baseTmpl = template.Must(FuncMap(template.New("base")).ParseFS(includeFS, "includes/*.tpl"))

	for _, class := range []Class{Charger, Meter, Vehicle, Tariff} {
		load(class)
	}
}

// Register adds a template file to the registry
func Register(class Class, filepath string) error {
	b, err := os.ReadFile(filepath)
	if err != nil {
		return err
	}

	tmpl, err := fromBytes(b)
	if err != nil {
		return fmt.Errorf("processing template '%s' failed: %w", filepath, err)
	}

	return register(class, tmpl)
}

func register(class Class, tmpl Template) error {
	if slices.ContainsFunc(templates[class], func(t Template) bool { return t.Template == tmpl.Template }) {
		return fmt.Errorf("duplicate template name: %s", tmpl.Template)
	}

	templates[class] = append(templates[class], tmpl)

	return nil
}

func fromBytes(b []byte) (Template, error) {
	// error on unknown fields
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)

	var definition TemplateDefinition
	if err := dec.Decode(&definition); err != nil {
		return Template{}, err
	}

	tmpl := Template{
		TemplateDefinition: definition,
	}

	for _, f := range []func() error{tmpl.ResolvePresets, tmpl.ResolveGroup, tmpl.UpdateParamsWithDefaults, tmpl.Validate} {
		if err := f(); err != nil {
			return tmpl, fmt.Errorf("template '%s': %w", tmpl.Template, err)
		}
	}

	return tmpl, nil
}

func load(class Class) {
	err := fs.WalkDir(definition.YamlTemplates, class.String(), func(filepath string, d fs.DirEntry, err error) error {
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

		tmpl, err := fromBytes(b)
		if err != nil {
			return fmt.Errorf("processing template '%s' failed: %w", filepath, err)
		}

		return register(class, tmpl)
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

type filterFunc func([]Template) []Template

// WithDeprecated returns a filterFunc that includes all templates
func WithDeprecated() filterFunc {
	return func(t []Template) []Template {
		return t
	}
}

// ByClass returns templates for class excluding deprecated templates
func ByClass(class Class, opt ...filterFunc) []Template {
	res := templates[class]
	if len(opt) == 0 {
		opt = append(opt, func(t []Template) []Template {
			return lo.Filter(t, func(t Template, _ int) bool {
				return !t.Deprecated
			})
		})
	}
	for _, o := range opt {
		res = o(res)
	}
	return res
}

// ByClass returns templates for class and name including deprecated templates
func ByName(class Class, name string) (Template, error) {
	for _, tmpl := range templates[class] {
		if tmpl.Template == name || slices.Contains(tmpl.Covers, name) {
			return tmpl, nil
		}
	}

	return Template{}, fmt.Errorf("template not found: %s", name)
}
