package mcp

import (
	"strconv"
	"strings"
)

func extractID(uri string) string {
	// Extract the ID from the URI, e.g., "loadpoints://123" -> "123"
	parts := strings.Split(uri, "://")
	if len(parts) < 2 {
		return ""
	}
	return parts[1]
}

func extractNumericID(uri string) int {
	s := extractID(uri)
	if s == "" {
		return 0
	}
	id, _ := strconv.Atoi(s)
	return id
}
