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

// String redacts a configuration string by replacing sensitive values with *****
func String(src string) string {
	return configRedactRegex.ReplaceAllString(src, "$1: *****")
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
