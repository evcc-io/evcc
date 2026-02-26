package homeassistant

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/samber/lo"
)

var log = util.NewLogger("homeassistant")

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /instances", getInstances)
	mux.HandleFunc("GET /entities", getEntities)
	mux.HandleFunc("GET /services", getServices)

	service.Register("homeassistant", mux)
}

func getInstances(w http.ResponseWriter, req *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	jsonWrite(w, slices.Sorted(maps.Values(instances)))
}

func connectionFromRequest(req *http.Request) (*Connection, error) {
	uri := util.DefaultScheme(strings.TrimSuffix(req.URL.Query().Get("uri"), "/"), "http")
	if uri == "" {
		return nil, errors.New("missing uri")
	}
	return NewConnection(log, uri, "")
}

func getEntities(w http.ResponseWriter, req *http.Request) {
	conn, err := connectionFromRequest(req)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	res, err := conn.GetStates()
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var domains []string
	if domain := req.URL.Query().Get("domain"); domain != "" {
		domains = strings.Split(domain, ",")
	}

	// cache list of entities
	w.Header().Set("Cache-control", "max-age=300")

	jsonWrite(w, lo.Map(lo.Filter(res, func(e StateResponse, _ int) bool {
		if len(domains) == 0 {
			return true
		}

		for _, domain := range domains {
			if strings.HasPrefix(e.EntityId, domain+".") {
				return true
			}
		}

		return false
	}), func(e StateResponse, _ int) string {
		return e.EntityId
	}))
}

type serviceDomainResponse struct {
	Domain   string         `json:"domain"`
	Services map[string]any `json:"services"`
}

func getServices(w http.ResponseWriter, req *http.Request) {
	conn, err := connectionFromRequest(req)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	var filterDomains []string
	if domain := req.URL.Query().Get("domain"); domain != "" {
		filterDomains = strings.Split(domain, ",")
	}

	seen := make(map[string]struct{})

	// collect matching services from /api/services
	var svcRes []serviceDomainResponse
	if err := conn.GetJSON(fmt.Sprintf("%s/api/services", conn.URI()), &svcRes); err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	for _, sd := range svcRes {
		if len(filterDomains) > 0 && !slices.Contains(filterDomains, sd.Domain) {
			continue
		}
		for svc := range sd.Services {
			seen[sd.Domain+"."+svc] = struct{}{}
		}
	}

	// collect matching entities from /api/states (e.g. Telegram notify entities)
	if states, err := conn.GetStates(); err == nil {
		for _, e := range states {
			for _, domain := range filterDomains {
				if strings.HasPrefix(e.EntityId, domain+".") {
					seen[e.EntityId] = struct{}{}
					break
				}
			}
		}
	}

	result := slices.Sorted(maps.Keys(seen))

	w.Header().Set("Cache-control", "max-age=300")
	jsonWrite(w, result)
}

// jsonWrite writes a JSON response
func jsonWrite(w http.ResponseWriter, data any) {
	json.NewEncoder(w).Encode(data)
}

// jsonError writes an error response
func jsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	jsonWrite(w, util.ErrorAsJson(err))
}
