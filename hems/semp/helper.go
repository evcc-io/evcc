package semp

import (
	"net"
)

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
