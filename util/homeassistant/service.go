package homeassistant

import (
	"encoding/json"
	"errors"
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
	mux.HandleFunc("GET /homes", getHomes)
	mux.HandleFunc("GET /homes/{home}/entities", getEntities)

	service.Register("homeassistant", mux)
}

func getHomes(w http.ResponseWriter, req *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	if len(instances) == 0 {
		jsonError(w, http.StatusPreconditionFailed, errors.New("no instances found"))
		return
	}

	type entry struct {
		Key string `json:"key"`
		Val string `json:"val"`
	}

	var res []entry
	for _, k := range slices.Sorted(maps.Keys(instances)) {
		res = append(res, entry{k, instances[k]})
	}

	jsonWrite(w, res)
}

func getEntities(w http.ResponseWriter, req *http.Request) {
	home := req.PathValue("home")

	if instanceUriByName(home) == "" {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn, _ := NewConnection(log, home)
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

// jsonWrite writes a JSON response
func jsonWrite(w http.ResponseWriter, data any) {
	json.NewEncoder(w).Encode(data)
}

// jsonError writes an error response
func jsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	jsonWrite(w, util.ErrorAsJson(err))
}
