package homeassistant

import (
	"encoding/json"
	"errors"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
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

	res := make([]service.Option, 0, len(instances))
	for name, uri := range instances {
		// match keeps the single-instance auto-fill
		res = append(res, service.Option{Value: uri, Label: name, Match: true})
	}

	jsonWrite(w, sortOptions(res))
}

func sortOptions(res []service.Option) []service.Option {
	slices.SortFunc(res, func(a, b service.Option) int {
		return strings.Compare(a.Value, b.Value)
	})
	return res
}

func entityOptions(states []StateResponse, domains, units []string) []service.Option {
	res := make([]service.Option, 0)
	for _, e := range states {
		if !matchesDomains(e.EntityId, domains) {
			continue
		}
		o := service.Option{
			Value: e.EntityId,
			Label: e.Attributes.FriendlyName,
			Match: slices.Contains(units, e.Attributes.UnitOfMeasurement),
		}
		if e.State != "unavailable" && e.State != "unknown" {
			o.Hint = strings.TrimSpace(e.State + " " + e.Attributes.UnitOfMeasurement)
		}
		res = append(res, o)
	}
	return sortOptions(res)
}

func connectionFromRequest(req *http.Request) (*Connection, error) {
	uri := util.DefaultScheme(strings.TrimSuffix(req.URL.Query().Get("uri"), "/"), "http")
	if uri == "" {
		return nil, errors.New("missing uri")
	}
	var insecure bool
	if s := req.URL.Query().Get("insecure"); s != "" {
		var err error
		if insecure, err = strconv.ParseBool(s); err != nil {
			return nil, err
		}
	}
	return NewConnection(log, uri, "", insecure)
}

// domainsFromRequest parses the comma-separated "domain" query parameter.
func domainsFromRequest(req *http.Request) []string {
	if domain := req.URL.Query().Get("domain"); domain != "" {
		return strings.Split(domain, ",")
	}
	return nil
}

// unitsFromRequest parses the comma-separated "unit" query parameter.
func unitsFromRequest(req *http.Request) []string {
	if unit := req.URL.Query().Get("unit"); unit != "" {
		return strings.Split(unit, ",")
	}
	return nil
}

// matchesDomains reports whether entityID belongs to any of the given domains.
// If domains is empty, all entities match.
func matchesDomains(entityID string, domains []string) bool {
	if len(domains) == 0 {
		return true
	}
	for _, d := range domains {
		if strings.HasPrefix(entityID, d+".") {
			return true
		}
	}
	return false
}

func getEntities(w http.ResponseWriter, req *http.Request) {
	conn, err := connectionFromRequest(req)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	states, err := conn.GetStates()
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	w.Header().Set("Cache-control", "max-age=300")
	jsonWrite(w, entityOptions(states, domainsFromRequest(req), unitsFromRequest(req)))
}

func getServices(w http.ResponseWriter, req *http.Request) {
	conn, err := connectionFromRequest(req)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	domains := domainsFromRequest(req)

	seen := make(map[string]struct{})

	// collect callable services from /api/services (e.g. notify.mobile_app_android)
	svcRes, err := conn.GetServices()
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}
	for _, sd := range svcRes {
		if len(domains) == 0 || slices.Contains(domains, sd.Domain) {
			for svc := range sd.Services {
				seen[sd.Domain+"."+svc] = struct{}{}
			}
		}
	}

	// collect entity-based notifiers from /api/states (e.g. Telegram in HA 2024+)
	if len(domains) > 0 {
		if states, err := conn.GetStates(); err == nil {
			for _, e := range states {
				if matchesDomains(e.EntityId, domains) {
					seen[e.EntityId] = struct{}{}
				}
			}
		}
	}

	w.Header().Set("Cache-control", "max-age=300")
	jsonWrite(w, slices.Sorted(maps.Keys(seen)))
}

// jsonWrite writes a JSON response
func jsonWrite(w http.ResponseWriter, data any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

// jsonError writes an error response
func jsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	jsonWrite(w, util.ErrorAsJson(err))
}
