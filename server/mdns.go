package server

import "github.com/grandcat/zeroconf"

// Announce evcc on a given HTTP port
func AnnounceMDNS(instance, service, host string, port int) (*zeroconf.Server, error) {
	return zeroconf.RegisterProxy(instance, service, "local.", port, host, nil, []string{}, nil)
}
