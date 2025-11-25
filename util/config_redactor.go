package util

//go:generate go run redact_gen.go

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// configRedactSecrets defines keys that should be redacted from configuration files
var configRedactSecrets []string

var configRedactRegex *regexp.Regexp

func init() {
	// fields that are not covered by template params (yet)
	additional := []string{
		"sponsortoken", "plant", // global settings
		"access", "refresh", "secret", // tokens not in params
		"deviceid", "machineid", "idtag", // devices
		"app", "chats", "recipients", // push messaging
	}

	// Combine generated params with additional fields
	configRedactSecrets = slices.Concat(generatedRedactParams, additional)

	configRedactRegex = regexp.MustCompile(fmt.Sprintf(`(?i)\b(%s)\b.*?:.*`, strings.Join(configRedactSecrets, "|")))
}

// RedactConfigString redacts a configuration string by replacing sensitive values with *****
func RedactConfigString(src string) string {
	return configRedactRegex.ReplaceAllString(src, "$1: *****")
}

// RedactConfigMap redacts sensitive keys in a configuration map
func RedactConfigMap(src map[string]any) map[string]any {
	res := maps.Clone(src)
	for k := range res {
		for _, secret := range configRedactSecrets {
			// ignore case
			if strings.EqualFold(k, secret) {
				res[k] = "*****"
				break
			}
		}
	}
	return res
}
