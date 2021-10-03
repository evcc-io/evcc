package custom

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/templates"
	"gopkg.in/yaml.v3"
)

type registry interface {
	Add(name string, factory func(map[string]interface{}) (api.Meter, error))
}

type instatiatorFunc func(typ string, other map[string]interface{}) (v api.Meter, err error)

func Register(registry registry, instantiator instatiatorFunc) {
	for _, tmpl := range templates.ByClass(templates.Meter) {
		println(strings.ToUpper(tmpl.Type))
		println("")

		// render the proxy
		sample, err := tmpl.RenderProxy()
		if err != nil {
			panic(err)
		}

		println("-- proxy --")
		println(string(sample))
		println("")

		instantiateFunc := instantiateFunction(tmpl, instantiator)
		registry.Add(tmpl.Type, instantiateFunc)

		// render all usages
		for _, usage := range tmpl.Usages() {
			println("--", usage, "--")

			b, err := tmpl.RenderResult(map[string]interface{}{
				"usage": usage,
			})
			if err != nil {
				panic(err)
			}

			println(string(b))
			println("")
		}
	}
}

func instantiateFunction(tmpl templates.Template, instantiator instatiatorFunc) func(map[string]interface{}) (api.Meter, error) {
	return func(other map[string]interface{}) (api.Meter, error) {
		b, err := tmpl.RenderResult(other)
		if err != nil {
			return nil, err
		}

		fmt.Println("-- instantiated --")
		println(string(b))
		println("")

		var instantiated struct {
			Type  string
			Other map[string]interface{} `yaml:",inline"`
		}

		if err := yaml.Unmarshal(b, &instantiated); err != nil {
			return nil, err
		}

		return instantiator(instantiated.Type, instantiated.Other)
	}
}
