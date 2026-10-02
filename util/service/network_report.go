package service

import (
	"net/http"
	"os"
	"runtime"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/discovery"
)

// reportDevice is shared by users, it must not identify the device or its owner
type reportDevice struct {
	Template  string   `json:"template"`
	Mac       string   `json:"mac,omitempty"`
	Hostnames []string `json:"hostnames,omitempty"`
	Services  []string `json:"services,omitempty"`
}

type scanHost struct {
	IP       string   `json:"ip"`
	Mac      string   `json:"mac,omitempty"`
	Vendor   string   `json:"vendor,omitempty"`
	Hostname string   `json:"hostname,omitempty"`
	Aliases  []string `json:"aliases,omitempty"`
	Services []string `json:"services,omitempty"`
	Used     bool     `json:"used,omitempty"`
}

// maskName cuts at the first digit, where serial numbers start
func maskName(s string) string {
	if idx := strings.IndexAny(s, "0123456789"); idx >= 0 {
		return s[:idx] + "*"
	}
	return s
}

func maskHostname(s string) string {
	name, _, _ := strings.Cut(s, ".")
	return maskName(name)
}

// maskService keeps mDNS types, cuts instance names and drops unique SSDP names
func maskService(s string) string {
	if strings.HasPrefix(s, "uuid:") {
		return ""
	}

	typ, instance, ok := strings.Cut(s, ":")
	if !ok || !strings.HasPrefix(s, "_") {
		return s
	}

	return typ + ":" + maskName(instance)
}

func system() string {
	res := runtime.GOOS + "/" + runtime.GOARCH

	if os.Getenv("SUPERVISOR_TOKEN") != "" {
		return res + " hassio"
	}
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return res + " docker"
	}

	return res
}

func reportDevices(configs []config.Named, hosts []discovery.Host) []reportDevice {
	res := make([]reportDevice, 0)

	for _, conf := range configs {
		name, ok := conf.Property("template").(string)
		if !ok || name == "" {
			continue
		}

		addrs := configHosts(conf)
		idx := slices.IndexFunc(hosts, func(h discovery.Host) bool {
			return isHost(h, addrs)
		})

		dev := reportDevice{Template: name}

		if idx >= 0 {
			h := hosts[idx]

			dev.Mac, _ = discovery.Registration(h.MAC)

			for _, name := range h.Names() {
				if name := maskHostname(name); !slices.Contains(dev.Hostnames, name) {
					dev.Hostnames = append(dev.Hostnames, name)
				}
			}

			for _, s := range h.Services {
				if s := maskService(s); s != "" && !slices.Contains(dev.Services, s) {
					dev.Services = append(dev.Services, s)
				}
			}
		}

		// one device may be configured multiple times
		if !slices.ContainsFunc(res, func(d reportDevice) bool {
			return d.Template == dev.Template && d.Mac == dev.Mac && slices.Equal(d.Hostnames, dev.Hostnames)
		}) {
			res = append(res, dev)
		}
	}

	return res
}

// report is shared by users. Hosts are unredacted and stay local.
type report struct {
	System  string         `json:"system"`
	Devices []reportDevice `json:"devices"`
	Hosts   []scanHost     `json:"hosts"`
}

func getReport(w http.ResponseWriter, req *http.Request) {
	found := hosts(w, req, mdnsTypes(allTemplates()))
	used := usedHosts()

	scanned := make([]scanHost, 0, len(found))
	for _, h := range found {
		scanned = append(scanned, scanHost{
			IP:       h.IP,
			Mac:      h.MAC,
			Vendor:   discovery.Vendor(h.MAC),
			Hostname: h.Hostname,
			Aliases:  h.Aliases,
			Services: h.Services,
			Used:     isHost(h, used),
		})
	}

	jsonWrite(w, report{
		System:  system(),
		Devices: reportDevices(allDeviceConfigs(), found),
		Hosts:   scanned,
	})
}
