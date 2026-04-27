package config

import (
	"github.com/spf13/cast"
	"go.yaml.in/yaml/v4"
)

// ParseEmbeddedDeviceYAML unmarshals a device's embedded YAML snippet and applies an optional top-level `type` override.
func ParseEmbeddedDeviceYAML(typ string, yamlStr string) (resolvedType string, cfg map[string]any, err error) {
	var res map[string]any
	if err = yaml.Unmarshal([]byte(yamlStr), &res); err != nil {
		return typ, nil, err
	}

	if override := cast.ToString(res["type"]); override != "" {
		delete(res, "type")
		return override, res, nil
	}

	return typ, res, nil
}

// CustomDevice promotes an embedded yaml type to the top-level type
func CustomDevice(typ string, other map[string]any) (string, map[string]any, error) {
	customYaml, ok := other["yaml"].(string)
	if !ok {
		return typ, other, nil
	}

	return ParseEmbeddedDeviceYAML(typ, customYaml)
}
