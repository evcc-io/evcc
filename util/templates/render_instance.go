package templates

import (
	"errors"
	"fmt"
	"os"

	"github.com/evcc-io/evcc/util"
	"gopkg.in/yaml.v3"
)

// Instance is an actual instantiated template
type Instance struct {
	Type  string
	Other map[string]interface{} `yaml:",inline"`
}

// RenderInstance renders an actual configuration instance
func RenderInstance(class Class, other map[string]interface{}) (*Instance, error) {
	var cc struct {
		Template string
		Other    map[string]interface{} `mapstructure:",remain"`
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	tmpl, err := ByName(class, cc.Template)
	if err != nil {
		return nil, err
	}

	b, _, err := tmpl.RenderResult(RenderModeInstance, other)
	if err != nil {
		return nil, util.NewConfigError(err)
	}

	if os.Getenv("EVCC_TEMPLATE_RENDER") == cc.Template {
		fmt.Println(string(b))
	}

	var instance Instance
	if err := yaml.Unmarshal(b, &instance); err != nil {
		return nil, fmt.Errorf("%w:\n%s", err, string(b))
	}

	if instance.Type == "" {
		return nil, errors.New("empty instance type- check for missing usage")
	}

	return &instance, nil
}
