package config

import (
	"sort"
	"strings"
)

type Type struct {
	Type   string `json:"type"`
	Label  string `json:"label"`
	Config interface{}
	Rank   int
}

var registry = make(map[string][]Type)

func Add(class string, types []Type) {
	registry[class] = types
}

type configType struct {
	Type   string       `json:"type"`
	Label  string       `json:"label"`
	Fields []Descriptor `json:"fields"`
}

func Types(class string) []configType {
	types := registry[class]

	sort.Slice(types, func(i, j int) bool {
		if types[i].Rank < types[j].Rank {
			return true
		}
		return strings.Compare(types[i].Type, types[j].Type) < 0
	})

	res := make([]configType, 0, len(types))

	for _, typ := range types {
		ct := configType{
			Type:   typ.Type,
			Label:  typ.Label,
			Fields: Describe(typ.Config),
		}
		res = append(res, ct)
	}

	return res
}
