package test

import (
	"strings"
)

// Acceptable checks if a test error is in the list of acceptable errors
func Acceptable(err error, acceptable []string) bool {
	for _, msg := range acceptable {
		err := strings.TrimSpace(err.Error())
		if strings.HasPrefix(err, msg) || strings.HasSuffix(err, msg) {
			return true
		}
	}

	return false
}
