package config

import (
	"fmt"
	"sort"
	"strings"
)

// Type is a registered type definition.
type Type struct {
	// Factory is duplicated here to allow creating devices by type without needing to import the device packages.
	Factory func(map[string]interface{}) (interface{}, error) `json:"-"`

	Type   string `json:"type"`
	Label  string `json:"label"`
	Config interface{}
	Rank   int
}

var registry = make(map[string][]Type)

// SetTypes sets the type definitions for given class in the registry
func SetTypes(class string, types []Type) {
	registry[class] = types
}

// typeDefinition retrieves type definitions by class and type
func typeDefinition(class, typ string) (Type, error) {
	types, ok := registry[class]
	if !ok {
		return Type{}, fmt.Errorf("invalid class: %s", class)
	}

	for _, v := range types {
		if v.Type == typ {
			return v, nil
		}
	}

	return Type{}, fmt.Errorf("invalid type: %s", typ)
}

// Types returns configuration types for given class
func Types(class string) []interface{} {
	types := registry[class]

	sort.Slice(types, func(i, j int) bool {
		if types[i].Rank < types[j].Rank {
			return true
		}
		return strings.ToLower(types[i].Type) < strings.ToLower(types[j].Type)
	})

	res := make([]interface{}, 0, len(types))

	for _, typ := range types {
		ct := description{
			Type:   typ.Type,
			Label:  typ.Label,
			Fields: prependType(typ.Type, annotate(typ.Config)),
		}
		res = append(res, ct)
	}

	return res
}
