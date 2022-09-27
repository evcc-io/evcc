package templates

import (
	"errors"
	"os"
	"testing"

	"github.com/evcc-io/evcc/util"
	"github.com/jinzhu/copier"
	"gopkg.in/yaml.v3"
)

// Instance is an actual instantiated template
type Instance struct {
	Type  string
	Other map[string]interface{} `yaml:",inline"`
}

// RenderInstance renders an actual configuration instance
func RenderInstance(class Class, other map[string]interface{}) (Instance, error) {
	var cc struct {
		Template string
		Other    map[string]interface{} `mapstructure:",remain"`
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return *new(Instance), err
	}

	tmpl, err := ByName(class, cc.Template)
	if err != nil {
		return *new(Instance), err
	}

	b, _, err := tmpl.RenderResult(TemplateRenderModeInstance, other)
	if err != nil {
		return *new(Instance), err
	}

	var instance Instance
	if err = yaml.Unmarshal(b, &instance); err == nil && instance.Type == "" {
		err = errors.New("empty instance type- check for missing usage")
	}

	return instance, err
}

// RenderTest renders and instantiates plus yaml-parses the template per usage
func RenderTest(t *testing.T, tmpl Template, values map[string]interface{}, cb func(values map[string]interface{})) {
	t.Run(tmpl.Template, func(t *testing.T) {
		t.Parallel()

		_, values, err := tmpl.RenderResult(TemplateRenderModeUnitTest, values)
		if err != nil {
			t.Log(tmpl.Render)
			t.Error(err)
		}

		// instantiate all usage variants
		for _, u := range tmpl.Usages() {
			t.Run(u, func(t *testing.T) {
				// t.Parallel()

				// create a copy of the map for parallel execution
				usageValues := make(map[string]interface{}, len(values)+1)
				if err := copier.Copy(&usageValues, values); err != nil {
					panic(err)
				}
				usageValues[ParamUsage] = u

				b, _, err := tmpl.RenderResult(TemplateRenderModeInstance, usageValues)
				if err != nil {
					t.Errorf("usage: %s, result: %v", u, err)
				}

				var instance interface{}
				if err := yaml.Unmarshal(b, &instance); err != nil {
					t.Errorf("usage: %s, yaml: %v", u, err)
				}

				// actually run the instance if not on CI
				if os.Getenv("CI") == "" {
					cb(usageValues)
				}
			})
		}
	})
}
