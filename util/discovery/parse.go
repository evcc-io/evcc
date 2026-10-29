package discovery

import (
	"net"
	"net/url"
	"regexp"
	"strings"
)

var neighborRE = regexp.MustCompile(`(\d{1,3}(?:\.\d{1,3}){3})\D.*?((?:[0-9A-Fa-f]{1,2}[:-]){5}[0-9A-Fa-f]{1,2})`)

// parseNeighbors handles /proc/net/arp and arp command output
func parseNeighbors(s string) map[string]string {
	res := make(map[string]string)

	for line := range strings.Lines(s) {
		m := neighborRE.FindStringSubmatch(line)
		if m == nil {
			continue
		}

		// macOS omits leading zeros, Windows separates by dash
		mac, err := net.ParseMAC(normalizeMAC(m[2]))
		if err != nil || isZeroOrBroadcast(mac) {
			continue
		}

		res[m[1]] = strings.ToUpper(mac.String())
	}

	return res
}

func normalizeMAC(s string) string {
	parts := strings.FieldsFunc(s, func(r rune) bool { return r == ':' || r == '-' })
	for i, p := range parts {
		if len(p) == 1 {
			parts[i] = "0" + p
		}
	}
	return strings.Join(parts, ":")
}

func isZeroOrBroadcast(mac net.HardwareAddr) bool {
	s := mac.String()
	return s == "00:00:00:00:00:00" || s == "ff:ff:ff:ff:ff:ff"
}

func locationHost(location string) string {
	u, err := url.Parse(location)
	if err != nil {
		return ""
	}
	return u.Hostname()
}
