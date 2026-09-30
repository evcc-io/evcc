// Package discovery finds hosts in the local network using unprivileged methods
package discovery

import (
	"cmp"
	"context"
	"maps"
	"net"
	"net/netip"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/evcc-io/evcc/util"
	"github.com/koron/go-ssdp"
	"github.com/korylprince/ipnetgen"
	"github.com/libp2p/zeroconf/v2"
)

const (
	cacheDuration = time.Minute
	retention     = 30 * time.Minute // devices miss scans
	lookupTimeout = time.Second
	minPrefixLen  = 24 // larger sweeps look like an attack
)

// quick first pass, second pass for late answers
var passes = []time.Duration{3 * time.Second, 5 * time.Second}

// Host is a device in the local network
type Host struct {
	IP       string   `json:"ip"`
	MAC      string   `json:"mac,omitempty"`
	Hostname string   `json:"hostname,omitempty"`
	Services []string `json:"services,omitempty"` // mDNS "type:instance", SSDP search target

	seen time.Time
}

var (
	log = util.NewLogger("discovery")

	mu       sync.Mutex
	hosts    = make(map[string]*Host)
	updated  time.Time     // end of last pass
	scanning bool          // scan is running
	passEnd  time.Time     // expected end of running pass
	ready    chan struct{} // closed after first pass of the scan
)

// Hosts returns the known hosts sorted by address. A scan runs in background
// when the result is outdated or refresh is set, the duration is its remaining time.
func Hosts(ctx context.Context, mdnsTypes []string, refresh bool) ([]Host, time.Duration) {
	mu.Lock()
	if !scanning && (refresh || time.Since(updated) > cacheDuration) {
		scanning = true
		ready = make(chan struct{})
		passEnd = time.Now().Add(passes[0] + lookupTimeout)
		go scan(ready, mdnsTypes)
	}
	wait := ready
	first := updated.IsZero()
	mu.Unlock()

	// nothing to show yet, wait for the first pass
	if first {
		select {
		case <-wait:
		case <-ctx.Done():
		}
	}

	mu.Lock()
	defer mu.Unlock()

	var pending time.Duration
	if scanning {
		pending = max(time.Second, time.Until(passEnd))
	}

	res := make([]Host, 0, len(hosts))
	for _, h := range hosts {
		res = append(res, *h)
	}

	slices.SortFunc(res, func(a, b Host) int {
		ia, _ := netip.ParseAddr(a.IP)
		ib, _ := netip.ParseAddr(b.IP)
		return cmp.Or(ia.Compare(ib), cmp.Compare(a.IP, b.IP))
	})

	return res, pending
}

func update(ip string, fun func(h *Host)) {
	addr, err := netip.ParseAddr(ip)
	if err != nil || !addr.Is4() || !addr.IsPrivate() {
		return
	}

	mu.Lock()
	defer mu.Unlock()

	h, ok := hosts[ip]
	if !ok {
		h = &Host{IP: ip}
		hosts[ip] = h
	}
	h.seen = time.Now()
	fun(h)
}

func addService(ip, service string) {
	update(ip, func(h *Host) {
		if !slices.Contains(h.Services, service) {
			h.Services = append(h.Services, service)
		}
	})
}

func scan(ready chan struct{}, mdnsTypes []string) {
	for i, duration := range passes {
		mu.Lock()
		passEnd = time.Now().Add(duration + lookupTimeout)
		mu.Unlock()

		pass(duration, mdnsTypes)

		mu.Lock()
		maps.DeleteFunc(hosts, func(_ string, h *Host) bool {
			return time.Since(h.seen) > retention
		})
		updated = time.Now()
		scanning = i < len(passes)-1
		mu.Unlock()

		// release requests waiting for the first pass
		if i == 0 {
			close(ready)
		}
	}
}

func pass(duration time.Duration, mdnsTypes []string) {
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	var wg sync.WaitGroup

	for _, typ := range slices.Compact(slices.Sorted(slices.Values(append(mdnsTypes, "_http._tcp")))) {
		wg.Go(func() { browse(ctx, typ) })
	}
	wg.Go(func() { search(duration) })
	wg.Go(func() {
		neighbors()
		warmup()

		// wait for address resolution
		<-ctx.Done()
		neighbors()
	})
	wg.Wait()

	lookupNames()
}

func localNetworks() []*net.IPNet {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.DEBUG.Println("interfaces:", err)
		return nil
	}

	var res []*net.IPNet
	for _, addr := range addrs {
		ipnet, ok := addr.(*net.IPNet)
		if !ok || ipnet.IP.To4() == nil || !ipnet.IP.IsPrivate() {
			continue
		}

		if ones, _ := ipnet.Mask.Size(); ones < minPrefixLen {
			mask := net.CIDRMask(minPrefixLen, 32)
			ipnet = &net.IPNet{IP: ipnet.IP.Mask(mask), Mask: mask}
		}

		res = append(res, ipnet)
	}

	return res
}

// warmup lets the operating system resolve all hardware addresses
func warmup() {
	for _, ipnet := range localNetworks() {
		gen, err := ipnetgen.New(ipnet.String())
		if err != nil {
			continue
		}

		for ip := gen.Next(); ip != nil; ip = gen.Next() {
			conn, err := net.DialUDP("udp4", nil, &net.UDPAddr{IP: ip, Port: 9})
			if err != nil {
				continue
			}
			_, _ = conn.Write(nil)
			conn.Close()
		}
	}
}

func neighbors() {
	entries, err := readNeighbors()
	if err != nil {
		log.DEBUG.Println("neighbors:", err)
		return
	}

	for ip, mac := range entries {
		update(ip, func(h *Host) { h.MAC = mac })
	}
}

func browse(ctx context.Context, typ string) {
	entries := make(chan *zeroconf.ServiceEntry, 16)

	go func() {
		for se := range entries {
			for _, ip := range se.AddrIPv4 {
				addService(ip.String(), typ+":"+se.Instance)
				update(ip.String(), func(h *Host) {
					// router name wins, mDNS may announce foreign addresses
					if h.Hostname == "" {
						h.Hostname = strings.TrimSuffix(se.HostName, ".")
					}
				})
			}
		}
	}()

	if err := zeroconf.Browse(ctx, typ, "local.", entries); err != nil {
		log.DEBUG.Println("mdns:", err)
		return
	}
	<-ctx.Done()
}

func search(duration time.Duration) {
	services, err := ssdp.Search(ssdp.All, int(duration.Seconds())-1, "")
	if err != nil {
		log.DEBUG.Println("ssdp:", err)
		return
	}

	for _, s := range services {
		if ip := locationHost(s.Location); ip != "" {
			addService(ip, s.Type)
		}
	}
}

func lookupNames() {
	mu.Lock()
	ips := slices.Collect(maps.Keys(hosts))
	mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), lookupTimeout)
	defer cancel()

	var wg sync.WaitGroup
	for _, ip := range ips {
		wg.Go(func() {
			names, err := net.DefaultResolver.LookupAddr(ctx, ip)
			if err != nil || len(names) == 0 {
				return
			}
			update(ip, func(h *Host) {
				h.Hostname = strings.TrimSuffix(names[0], ".")
			})
		})
	}
	wg.Wait()
}
