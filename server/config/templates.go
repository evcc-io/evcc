package config

import (
	"fmt"

	"github.com/andig/evcc/util/test"
)

type configSample = struct {
	Name   string `json:"name"`
	Sample string `json:"template"`
}

// SamplesByClass returns a slice of configuration templates
func SamplesByClass(class string) []configSample {
	res := make([]configSample, 0)
	for _, conf := range test.ConfigTemplates(class) {
		typedSample := fmt.Sprintf("type: %s\n%s", conf.Type, conf.Sample)
		t := configSample{
			Name:   conf.Name,
			Sample: typedSample,
		}
		res = append(res, t)
	}
	return res
}
