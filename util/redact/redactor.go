package redact

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util/templates"
	"github.com/samber/lo"
)

var (
	configRedactRegex   *regexp.Regexp
	configRedactSecrets []string
)

func init() {
	// fields that are not covered by template params (yet)
	additional := []string{
		"sponsortoken", "plant", // global settings
		"app", "chats", "recipients", // push messaging
	}

	// Combine generated params with additional fields
	configRedactSecrets = slices.Concat(redactableParams(), additional)

	configRedactRegex = regexp.MustCompile(fmt.Sprintf(`(?i)\b(%s)\b.*?:.*`, strings.Join(configRedactSecrets, "|")))
}

func redactableParams() []string {
	// Collect all sensitive params from templates (includes defaults)
	var params []string
	for _, class := range templates.ClassValues() {
		for _, tmpl := range templates.ByClass(class) {
			for _, p := range tmpl.Params {
				if p.IsMasked() || p.IsPrivate() {
					params = append(params, strings.ToLower(p.Name))
				}
			}
		}
	}

	return lo.Uniq(params)
}

// String redacts a configuration string by replacing sensitive values with *****.
// Lines nested below a redacted key (block scalars, nested maps, lists) are dropped.
func String(src string) string {
	lines := strings.Split(src, "\n")
	res := make([]string, 0, len(lines))

	skip := -1 // indent of the redacted key whose nested lines are dropped
	for _, line := range lines {
		indent := len(line) - len(strings.TrimLeft(line, " \t"))
		if skip >= 0 && (indent > skip || strings.TrimSpace(line) == "") {
			continue
		}

		skip = -1
		if redacted := configRedactRegex.ReplaceAllString(line, "$1: *****"); redacted != line {
			line = redacted
			skip = indent
		}

		res = append(res, line)
	}

	return strings.Join(res, "\n")
}

// Map redacts sensitive keys in a configuration map
func Map(src map[string]any) map[string]any {
	res := maps.Clone(src)
	for k := range res {
		if slices.ContainsFunc(configRedactSecrets, func(s string) bool {
			return strings.EqualFold(k, s)
		}) {
			res[k] = "*****"
		}
	}
	return res
}
