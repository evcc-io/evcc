package test

import (
	"strings"
)

const config = "../errors.yaml"

var acceptable map[string][]string

// Acceptable checks if a test error is configured as acceptable
func Acceptable(err error, acceptable []string) bool {
	for _, msg := range acceptable {
		if strings.HasPrefix(err.Error(), msg) || strings.HasSuffix(err.Error(), msg) {
			return true
		}
	}

	return false
}
