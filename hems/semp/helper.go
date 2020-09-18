package semp

import "net"

// LocalIPs returns a slice of local IPv4 addresses
func LocalIPs() []net.IP {
	ips := make([]net.IP, 0)

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
			if ip, ok := addr.(*net.IPNet); ok {
				if !ip.IP.IsLoopback() && ip.IP.To4() != nil {
					ips = append(ips, ip.IP)
				}
			}
		}
	}

	return ips
}
