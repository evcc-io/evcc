package config

import (
	"strings"
)

type Typed struct {
	Type  string         `json:"type"`
	Other map[string]any `mapstructure:",remain" yaml:",inline"`
}

type Named struct {
	Name  string         `json:"name"`
	Type  string         `json:"type"`
	Other map[string]any `mapstructure:",remain" yaml:",inline"`
}

// Property returns the value of the named property
func (n Named) Property(key string) any {
	for k, v := range n.Other {
		if strings.EqualFold(k, key) {
			return v
		}
	}
	return nil
}
