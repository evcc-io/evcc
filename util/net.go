package util

import (
	"fmt"
	"net"
)

// DefaultPort appends given port to connection if not specified
func DefaultPort(conn string, port int) string {
	if _, _, err := net.SplitHostPort(conn); err != nil {
		conn = fmt.Sprintf("%s:%d", conn, port)
	}

	return conn
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
