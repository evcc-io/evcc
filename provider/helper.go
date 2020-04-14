package provider

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var re = regexp.MustCompile(`\${(\w+)(:([a-zA-Z0-9%.]+))?}`)

// truish returns true if value is truish (true/1/on)
func truish(s string) bool {
	return s == "1" || strings.ToLower(s) == "true" || strings.ToLower(s) == "on"
}

// formatValue will apply specific formatting in addition to standard sprintf
func formatValue(format string, val interface{}) string {
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

// replaceFormatted replaces all occurrences of ${key} with formatted val from the kv map
func replaceFormatted(s string, kv map[string]interface{}) (string, error) {
	for m := re.FindStringSubmatch(s); m != nil; m = re.FindStringSubmatch(s) {
		// find key and replacement value
		val, ok := kv[m[1]]
		if !ok {
			return "", errors.New("could find value for: " + m[0])
		}

		// update all literal matches
		new := formatValue(m[3], val)
		s = strings.ReplaceAll(s, m[0], new)
	}

	return s, nil
}
