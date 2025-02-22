package config

import (
	"strings"
)

type Typed struct {
	Type  string         `json:"type"`
	Other map[string]any `mapstructure:",remain" yaml:",inline"`

	Title              string `json:"title"`
	Icon               string `json:"icon"`
	ProductBrand       string `json:"productBrand"`
	ProductDescription string `json:"productDescription"`
}

type Named struct {
	Name  string `json:"name"`
	Typed `mapstructure:",squash" yaml:",inline"`
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
