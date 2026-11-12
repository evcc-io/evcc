package homeassistant

import (
	"cmp"
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

	slices.SortFunc(res, func(a, b service.Option) int { return cmp.Compare(a.Value, b.Value) })
	jsonWrite(w, res)
}

func entityOptions(states []StateResponse, domains, units []string) []service.Option {
	var res []service.Option
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
	slices.SortFunc(res, func(a, b service.Option) int { return cmp.Compare(a.Value, b.Value) })
	return res
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

// queryList parses a comma-separated query parameter.
func queryList(req *http.Request, key string) []string {
	if v := req.URL.Query().Get(key); v != "" {
		return strings.Split(v, ",")
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

	// short cache: hints carry live values, the form refetches on every change
	w.Header().Set("Cache-control", "max-age=30")
	jsonWrite(w, entityOptions(states, queryList(req, "domain"), queryList(req, "unit")))
}

func getServices(w http.ResponseWriter, req *http.Request) {
	conn, err := connectionFromRequest(req)
	if err != nil {
		jsonError(w, http.StatusBadRequest, err)
		return
	}

	domains := queryList(req, "domain")

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
