package config

import (
	"github.com/evcc-io/evcc/util/yaml"
	"github.com/spf13/cast"
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

	// type override
	if override := cast.ToString(res["type"]); override != "" {
		delete(res, "type")
		return override, res, nil
	}

	return typ, res, nil
}
