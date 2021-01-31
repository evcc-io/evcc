package util

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var re = regexp.MustCompile(`\${(\w+)(:([a-zA-Z0-9%.]+))?}`)

// Truish returns true if value is truish (true/1/on)
func Truish(s string) bool {
	return s == "1" || strings.ToLower(s) == "true" || strings.ToLower(s) == "on"
}

// FormatValue will apply specific formatting in addition to standard sprintf
func FormatValue(format string, val interface{}) string {
	switch typed := val.(type) {
	case bool:
		if format == "%d" {
			if typed {
				return "1"
			}
			return "0"
		}
	case float64:
		switch {
		case strings.HasSuffix(format, "m"): // milli
			format = format[:len(format)-1]
			val = typed * 1e3
		case strings.HasSuffix(format, "k"): // kilo
			format = format[:len(format)-1]
			val = typed / 1e3
		}
	case time.Duration:
		val = typed.Round(time.Second)
	}

	if format == "" {
		format = "%v"
	}

	return fmt.Sprintf(format, val)
}

// ReplaceFormatted replaces all occurrences of ${key} with formatted val from the kv map
func ReplaceFormatted(s string, kv map[string]interface{}) (string, error) {
	wanted := make([]string, 0)

	for m := re.FindStringSubmatch(s); m != nil; m = re.FindStringSubmatch(s) {
		match, key, format := m[0], m[1], m[3]

		// find key and replacement value
		val, ok := kv[key]
		if !ok {
			wanted = append(wanted, key)
			format = "%s"
			val = "?"
		}

		// update all literal matches
		new := FormatValue(format, val)
		s = strings.ReplaceAll(s, match, new)
	}

	// return missing keys
	var err error
	if len(wanted) > 0 {
		got := make([]string, 0)
		for k := range kv {
			got = append(got, k)
		}

		err = fmt.Errorf("wanted: %v, got: %v", wanted, got)
	}

	return s, err
}
