package util

import (
	"fmt"
	"maps"
	"regexp"
	"slices"
	"strings"
)

// configRedactSecrets defines keys that should be redacted from configuration files
var configRedactSecrets = []string{
	"mac",                   // infrastructure
	"sponsortoken", "plant", // global settings
	"apikey", "user", "password", "pin", // users
	"token", "access", "refresh", "accesstoken", "refreshtoken", // tokens, including template variations
	"ain", "secret", "serial", "deviceid", "machineid", "idtag", // devices
	"app", "chats", "recipients", // push messaging
	"vin",               // vehicles
	"lat", "lon", "zip", // solar forecast
}

var configRedactRegex = regexp.MustCompile(fmt.Sprintf(`(?i)\b(%s)\b.*?:.*`, strings.Join(configRedactSecrets, "|")))

// RedactConfigString redacts a configuration string by replacing sensitive values with *****
func RedactConfigString(src string) string {
	return configRedactRegex.ReplaceAllString(src, "$1: *****")
}

// RedactConfigMap redacts sensitive keys in a configuration map
func RedactConfigMap(src map[string]any) map[string]any {
	res := maps.Clone(src)
	for k := range res {
		if slices.Contains(configRedactSecrets, k) {
			res[k] = "*****"
		}
	}
	return res
}
