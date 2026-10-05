package service

import (
	"context"
	"encoding/json"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/discovery"
	"github.com/evcc-io/evcc/util/templates"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /hosts", getHosts)
	mux.HandleFunc("GET /report", getReport)

	service.Register("network", mux)
}

var ipRE = regexp.MustCompile(`\d{1,3}(\.\d{1,3}){3}`)

func allTemplates() []templates.Template {
	var res []templates.Template
	for _, class := range templates.ClassValues() {
		res = append(res, templates.ByClass(class)...)
	}
	return res
}

func deviceConfigs[T any](handler config.Handler[T]) []config.Named {
	var res []config.Named
	for _, dev := range handler.Devices() {
		res = append(res, dev.Config())
	}
	return res
}

func allDeviceConfigs() []config.Named {
	return slices.Concat(
		deviceConfigs(config.Chargers()),
		deviceConfigs(config.Meters()),
		deviceConfigs(config.Vehicles()),
		deviceConfigs(config.Messengers()),
		deviceConfigs(config.Curtailers()),
	)
}

// configHosts returns lower case names and addresses of the device config
func configHosts(conf config.Named) []string {
	var res []string
	for k, v := range conf.Other {
		s, ok := v.(string)
		if !ok {
			continue
		}

		res = append(res, strings.ToLower(s))
		res = append(res, ipRE.FindAllString(s, -1)...)

		if slices.Contains([]string{"host", "ip", "uri"}, strings.ToLower(k)) {
			res = append(res, resolve(s)...)
		}
	}
	return res
}

func resolve(s string) []string {
	host := s
	if u, err := url.Parse(util.DefaultScheme(s, "http")); err == nil && u.Hostname() != "" {
		host = u.Hostname()
	}

	if net.ParseIP(host) != nil {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	addrs, _ := net.DefaultResolver.LookupHost(ctx, host)
	return addrs
}

func isHost(h discovery.Host, hosts []string) bool {
	return slices.Contains(hosts, h.IP) || slices.ContainsFunc(h.Names(), func(name string) bool {
		return slices.Contains(hosts, strings.ToLower(name))
	})
}

func usedHosts(configs []config.Named) []string {
	var res []string
	for _, conf := range configs {
		res = append(res, configHosts(conf)...)
	}
	return res
}

func hosts(w http.ResponseWriter, req *http.Request, mdnsTypes []string) []discovery.Host {
	// static list instead of scanning, e.g. for containers without access to the network
	if env := os.Getenv("EVCC_DISCOVERY_HOSTS"); env != "" {
		var res []discovery.Host
		_ = json.Unmarshal([]byte(env), &res)
		return res
	}

	res, pending := discovery.Hosts(req.Context(), mdnsTypes, req.URL.Query().Has("refresh"))

	// scan still running, instruct client to fetch again
	if pending > 0 {
		w.Header().Set("Retry-After", strconv.Itoa(int(math.Ceil(pending.Seconds()))))
	}

	return res
}

func mdnsTypes(all []templates.Template) []string {
	var res []string
	for _, t := range all {
		res = append(res, t.Discovery.MdnsTypes()...)
	}
	return res
}

func discoveryByName(all []templates.Template, name string) []templates.Discovery {
	var res []templates.Discovery
	for _, t := range all {
		if t.Template == name && name != "" {
			res = append(res, t.Discovery)
		}
	}
	return res
}

func getHosts(w http.ResponseWriter, req *http.Request) {
	all := allTemplates()
	selected := discoveryByName(all, req.URL.Query().Get("template"))

	used := usedHosts(allDeviceConfigs())

	res := make([]Option, 0)
	for _, h := range hosts(w, req, mdnsTypes(all)) {
		match := func(d templates.Discovery) bool {
			return d.Match(h.Names(), h.MAC, h.Services)
		}

		o := Option{
			Value: h.IP,
			Label: h.Hostname,
			Hint:  discovery.Vendor(h.MAC),
			Match: slices.ContainsFunc(selected, match),
			Used:  isHost(h, used),
		}

		// template brand beats hardware vendor
		if idx := slices.IndexFunc(all, func(t templates.Template) bool {
			return len(t.Products) > 0 && t.Products[0].Brand != "" && match(t.Discovery)
		}); idx >= 0 {
			o.Hint = all[idx].Products[0].Brand
		}

		res = append(res, o)
	}

	jsonWrite(w, res)
}
