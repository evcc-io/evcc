package meter

import (
	"testing"

	"github.com/andig/evcc-config/registry"
	_ "github.com/andig/evcc-config/templates"
	"github.com/andig/evcc/util"
	"gopkg.in/yaml.v3"
)

func TemplatesByClass(class string) []registry.Template {
	templates := make([]registry.Template, 0)
	for _, t := range registry.Registry {
		if t.Class == class {
			templates = append(templates, t)
		}
	}
	return templates
}

func TestMeters(t *testing.T) {
	log := util.NewLogger("foo")
	templates := TemplatesByClass("meter")

	for _, tmpl := range templates {
		t.Logf("%s: %s", tmpl.Class, tmpl.Name)

		var conf map[string]interface{}
		if err := yaml.Unmarshal([]byte(tmpl.Sample), &conf); err != nil {
			t.Error(err)
		}

		_ = NewFromConfig(log, tmpl.Type, conf)
	}
}
