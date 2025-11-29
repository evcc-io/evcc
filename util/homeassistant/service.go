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
	mux.HandleFunc("GET /instances", getInstances)
	mux.HandleFunc("GET /entities", getEntities)

	service.Register("homeassistant", mux)
}

func getInstances(w http.ResponseWriter, req *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	jsonWrite(w, slices.Sorted(maps.Values(instances)))
}

func getEntities(w http.ResponseWriter, req *http.Request) {
	uri := req.URL.Query().Get("uri")
	if uri == "" {
		jsonError(w, http.StatusBadRequest, errors.New("missing uri"))
		return
	}

	conn, _ := NewConnection(log, uri, "")
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
