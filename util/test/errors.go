package test

import (
	"strings"
)

// Acceptable checks if a test error is in the list of acceptable errors
func Acceptable(err error, acceptable []string) bool {
	for _, msg := range acceptable {
		if strings.HasPrefix(err.Error(), msg) || strings.HasSuffix(err.Error(), msg) {
			return true
		}
	}

	return false
}
