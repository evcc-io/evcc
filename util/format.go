package util

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var re = regexp.MustCompile(`\${(\w+)(:([a-zA-Z0-9%.]+))?}`)

// Truish returns true if value is truish (true/1/on)
func Truish(s string) bool {
	return s == "1" || strings.ToLower(s) == "true" || strings.ToLower(s) == "on"
}

// FormatValue will apply specific formatting in addition to standard sprintf
func FormatValue(format string, val interface{}) string {
	switch val := val.(type) {
	case bool:
		if format == "%d" {
			if val {
				return "1"
			}
			return "0"
		}
	}

	if format == "" {
		format = "%v"
	}

	return fmt.Sprintf(format, val)
}

// ReplaceFormatted replaces all occurrences of ${key} with formatted val from the kv map
func ReplaceFormatted(s string, kv map[string]interface{}) (string, error) {
	for m := re.FindStringSubmatch(s); m != nil; m = re.FindStringSubmatch(s) {
		// find key and replacement value
		val, ok := kv[m[1]]
		if !ok {
			return "", errors.New("could find value for: " + m[0])
		}

		// update all literal matches
		new := FormatValue(m[3], val)
		s = strings.ReplaceAll(s, m[0], new)
	}

	return s, nil
}
