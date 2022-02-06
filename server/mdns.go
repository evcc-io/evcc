package server

import "github.com/grandcat/zeroconf"

// Announce evcc on a given HTTP port
func AnnounceMDNS(instance, service, host string, port int) (*zeroconf.Server, error) {
	return zeroconf.RegisterProxy("evcc Website", "_http._tcp", "local.", port, "evcc", nil, []string{}, nil)
}
