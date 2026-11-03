package templates

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Discovery describes how devices are recognized in the local network
type Discovery struct {
	Mdns     []string // service type as announced, with optional instance pattern: _http._tcp:shelly*
	Hostname []string // hostname pattern: sma*
	Mac      []string // hardware address prefix: 0015BB
}

var (
	mdnsRE = regexp.MustCompile(`^_[A-Za-z0-9_-]+\._(tcp|udp)(:.+)?$`)
	macRE  = regexp.MustCompile(`^[0-9A-F]{6,9}$`)
)

func (d Discovery) validate() error {
	for _, s := range d.Mdns {
		if !mdnsRE.MatchString(s) {
			return fmt.Errorf("invalid discovery mdns: '%s'", s)
		}
	}
	for _, s := range d.Mac {
		if !macRE.MatchString(s) {
			return fmt.Errorf("invalid discovery mac: '%s'", s)
		}
	}
	for _, s := range append(d.Hostname, d.Mdns...) {
		if _, err := path.Match(s, ""); err != nil {
			return fmt.Errorf("invalid discovery pattern: '%s'", s)
		}
	}
	return nil
}

// MdnsTypes returns the service types to browse
func (d Discovery) MdnsTypes() []string {
	res := make([]string, 0, len(d.Mdns))
	for _, s := range d.Mdns {
		typ, _, _ := strings.Cut(s, ":")
		res = append(res, typ)
	}
	return res
}

// Match returns true if any hint applies. Services are mDNS "type:instance".
func (d Discovery) Match(hostnames []string, mac string, services []string) bool {
	hex := strings.ToUpper(strings.NewReplacer(":", "", "-", "").Replace(mac))
	for _, prefix := range d.Mac {
		if strings.HasPrefix(hex, prefix) {
			return true
		}
	}

	for _, hostname := range hostnames {
		// domain is not part of the device name
		name, _, _ := strings.Cut(strings.ToLower(hostname), ".")
		for _, pattern := range d.Hostname {
			if ok, _ := path.Match(strings.ToLower(pattern), name); ok && name != "" {
				return true
			}
		}
	}

	for _, service := range services {
		for _, s := range d.Mdns {
			if !strings.Contains(s, ":") {
				s += ":*"
			}
			if ok, _ := path.Match(strings.ToLower(s), strings.ToLower(service)); ok {
				return true
			}
		}
	}

	return false
}
