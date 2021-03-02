package util

import (
	"fmt"
	"net"
	"net/url"
)

// DefaultPort appends given port to connection if not specified
func DefaultPort(conn string, port int) string {
	if _, _, err := net.SplitHostPort(conn); err != nil {
		conn = fmt.Sprintf("%s:%d", conn, port)
	}

	return conn
}

// DefaultScheme prepends given scheme to uri if not specified
func DefaultScheme(uri string, scheme string) string {
	u, err := url.Parse(uri)
	if err != nil {
		return uri
	}

	switch {
	case u.Scheme == "":
		// scheme missing
		u.Scheme = scheme

	case u.Opaque != "":
		// host:port format is parsed as scheme:opaque (https://golang.org/pkg/net/url/#URL)
		if u, err = url.Parse(fmt.Sprintf("%s://%s", scheme, uri)); err != nil {
			return uri
		}
	}

	return u.String()
}

// LocalIPs returns a slice of local IPv4 addresses
func LocalIPs() (ips []net.IPNet) {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	for _, i := range ifaces {
		addrs, err := i.Addrs()
		if err != nil {
			panic(err)
		}

		for _, addr := range addrs {
			// fmt.Println(addr)
			if ip, ok := addr.(*net.IPNet); ok {
				if !ip.IP.IsLoopback() && ip.IP.To4() != nil {
					ips = append(ips, *ip)
				}
			}
		}
	}

	return ips
}
