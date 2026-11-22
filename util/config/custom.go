package config

import (
	"github.com/spf13/cast"
	"go.yaml.in/yaml/v4"
)

// CustomDevice promotes an embedded yaml type to the top-level type
func CustomDevice(typ string, other map[string]any) (string, map[string]any, error) {
	customYaml, ok := other["yaml"].(string)
	if !ok {
		return typ, other, nil
	}

	var res map[string]any
	if err := yaml.Unmarshal([]byte(customYaml), &res); err != nil {
		return typ, nil, err
	}
	if res == nil {
		res = make(map[string]any)
	}

	// structured fields stored next to the yaml take precedence
	for k, v := range other {
		if k != "yaml" {
			res[k] = v
		}
	}

	// type override
	if override := cast.ToString(res["type"]); override != "" {
		delete(res, "type")
		return override, res, nil
	}

	return typ, res, nil
}
