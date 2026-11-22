package discovery

import (
	"net"
	"strings"
	"syscall"

	"golang.org/x/net/route"
)

// the arp command returns nothing when called from a service
func readNeighbors() (map[string]string, error) {
	rib, err := route.FetchRIB(syscall.AF_INET, syscall.NET_RT_FLAGS, syscall.RTF_LLINFO)
	if err != nil {
		return nil, err
	}

	msgs, err := route.ParseRIB(syscall.NET_RT_FLAGS, rib)
	if err != nil {
		return nil, err
	}

	res := make(map[string]string)

	for _, m := range msgs {
		rm, ok := m.(*route.RouteMessage)
		if !ok || len(rm.Addrs) <= syscall.RTAX_GATEWAY {
			continue
		}

		ip, ok := rm.Addrs[syscall.RTAX_DST].(*route.Inet4Addr)
		if !ok {
			continue
		}

		ll, ok := rm.Addrs[syscall.RTAX_GATEWAY].(*route.LinkAddr)
		if !ok || len(ll.Addr) != 6 {
			continue
		}

		if mac := net.HardwareAddr(ll.Addr); !isZeroOrBroadcast(mac) {
			res[net.IP(ip.IP[:]).String()] = strings.ToUpper(mac.String())
		}
	}

	return res, nil
}
