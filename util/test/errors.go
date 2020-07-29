package test

import (
	"io/ioutil"
	"strings"

	"gopkg.in/yaml.v3"
)

const config = "../errors.yaml"

var acceptable map[string][]string

// Acceptable returns true is a test error is configured as acceptable
func Acceptable(class string, err error) bool {
	if len(acceptable) == 0 {
		definitions, err := ioutil.ReadFile(config)
		if err != nil {
			panic(err)
		}
		if err := yaml.Unmarshal(definitions, &acceptable); err != nil {
			panic(err)
		}
	}

	for _, msg := range acceptable[class] {
		if strings.HasPrefix(err.Error(), msg) {
			return true
		}
	}

	return false
}
