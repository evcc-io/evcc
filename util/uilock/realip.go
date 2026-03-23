package uilock

import (
	"net"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/util"
)

var log = util.NewLogger("uilock")

// EffectiveClientIP returns the client IP for access control. If trustedProxies is non-empty
// and the direct TCP peer matches one of those CIDRs, the IP is taken from X-Forwarded-For
// (leftmost valid hop) or X-Real-IP; otherwise RemoteAddr is used and forwarded headers are ignored.
func EffectiveClientIP(r *http.Request, trustedProxies []*net.IPNet) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	direct := net.ParseIP(host)
	if direct == nil {
		return nil
	}

	if len(trustedProxies) == 0 || !ipInNets(direct, trustedProxies) {
		return direct
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			// IPv6 in XFF can be bracketed
			p = strings.TrimPrefix(strings.TrimSuffix(p, "]"), "[")
			if ip := net.ParseIP(p); ip != nil {
				return ip
			}
		}
	}

	if xr := strings.TrimSpace(r.Header.Get("X-Real-IP")); xr != "" {
		if ip := net.ParseIP(xr); ip != nil {
			return ip
		}
	}

	return direct
}

func ipInNets(ip net.IP, nets []*net.IPNet) bool {
	for _, n := range nets {
		if n.Contains(ip) {
			return true
		}
	}
	return false
}

// ParseCIDRList parses CIDR strings. Plain IPs are normalized to /32 or /128.
// Invalid entries are skipped with a warning log.
func ParseCIDRList(ss []string) []*net.IPNet {
	var out []*net.IPNet
	for _, s := range ss {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		if !strings.Contains(s, "/") {
			if ip := net.ParseIP(s); ip != nil {
				if ip4 := ip.To4(); ip4 != nil {
					s = ip4.String() + "/32"
				} else {
					s = ip.String() + "/128"
				}
			}
		}
		_, n, err := net.ParseCIDR(s)
		if err != nil {
			log.WARN.Printf("ignoring invalid CIDR entry %q: %v", s, err)
			continue
		}
		out = append(out, n)
	}
	return out
}
