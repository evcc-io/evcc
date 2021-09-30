package meter

import (
	"fmt"
	"strings"

	public "github.com/evcc-io/config/registry"
	"github.com/evcc-io/evcc/api"
	"gopkg.in/yaml.v2"
)

type meterRegistry map[string]func(map[string]interface{}) (api.Meter, error)

func (r meterRegistry) Add(name string, factory func(map[string]interface{}) (api.Meter, error)) {
	if _, exists := r[name]; exists {
		panic(fmt.Sprintf("cannot register duplicate meter type: %s", name))
	}
	r[name] = factory
}

func (r meterRegistry) Get(name string) (func(map[string]interface{}) (api.Meter, error), error) {
	factory, exists := r[name]
	if !exists {
		return nil, fmt.Errorf("meter type not registered: %s", name)
	}
	return factory, nil
}

var registry meterRegistry = make(map[string]func(map[string]interface{}) (api.Meter, error))

// NewFromConfig creates meter from configuration
func NewFromConfig(typ string, other map[string]interface{}) (v api.Meter, err error) {
	if typ == "custom" && other["public"] != nil {
		// this is a custom configuration where we need to load the proper configuration from a generated config class
		template, err := public.TemplateByPublic("meter", strings.ToLower(typ), strings.ToLower(other["public"].(string)))
		if err == nil {
			// insert all param values into the template Params section
			for index, param := range template.Params {
				paramFound := false
				for key, value := range other {
					if key == "public" {
						continue
					}
					if param.Name == key {
						template.Params[index].Value = value.(string)
						paramFound = true
						break
					}
				}
				if !paramFound {
					return nil, fmt.Errorf("param '%s' is missing!", param.Name)
				}
			}
		}
		// create a config via the template
		template = template.RenderSample()

		// parse the meter config
		other = make(map[string]interface{})
		if err := yaml.Unmarshal([]byte(template.Sample), &other); err != nil {
			return nil, fmt.Errorf("failed parsing template %s", template.Name)
		}
	}

	factory, err := registry.Get(strings.ToLower(typ))
	if err == nil {
		if v, err = factory(other); err != nil {
			err = fmt.Errorf("cannot create meter '%s': %w", typ, err)
		}
	} else {
		err = fmt.Errorf("invalid meter type: %s", typ)
	}

	return
}
