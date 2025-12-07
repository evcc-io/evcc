package util

import (
	"bytes"
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
	"text/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/samber/lo"
)

var re = regexp.MustCompile(`(?i)\${(\w+)(:([a-zA-Z0-9%.]+))?}`)

// FormatValue will apply specific formatting in addition to standard sprintf
func FormatValue(format string, val any) string {
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
func ReplaceFormatted(s string, kv map[string]any) (string, error) {
	// Enhanced golang template logic
	tpl, err := template.New("base").
		Funcs(sprig.TxtFuncMap()).
		Funcs(map[string]any{
			"timeRound": timeRound,
			"addDate":   addDate,
		}).Parse(s)
	if err != nil {
		return s, err
	}

	var rs bytes.Buffer
	if err := tpl.Execute(&rs, kv); err != nil {
		return s, err
	}
	s = rs.String()

	// Regex logic for backward compatibility
	wanted := make([]string, 0)

	for m := re.FindStringSubmatch(s); m != nil; m = re.FindStringSubmatch(s) {
		match, key, format := m[0], m[1], m[3]

		// find key and replacement value
		var val *any
		for k, v := range kv {
			if strings.EqualFold(k, key) {
				val = &v
				break
			}
		}

		if val == nil {
			wanted = append(wanted, key)
			format = "%s"
			val = lo.ToPtr(any("?"))
		}

		// update all literal matches
		new := FormatValue(format, *val)
		s = strings.ReplaceAll(s, match, new)
	}

	// return missing keys
	if len(wanted) > 0 {
		return "", fmt.Errorf("wanted: %v, got: %v", slices.Sorted(slices.Values(wanted)), slices.Sorted(maps.Keys(kv)))
	}

	return s, nil
}
