package eebus

import "strings"

func NormalizeSki(ski string) string {
	ski = strings.ReplaceAll(ski, "-", "")
	ski = strings.ReplaceAll(ski, " ", "")
	return strings.ToLower(ski)
}
