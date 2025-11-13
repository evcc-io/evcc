package homeassistant

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"

	"github.com/evcc-io/evcc/server/service"
	"github.com/evcc-io/evcc/util"
	"github.com/samber/lo"
)

var log = util.NewLogger("homeassistant")

func init() {
	handler := new(handler)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /homes", handler.Homes)
	mux.HandleFunc("GET /homes/{home}/entities", handler.Home)

	service.Register("homeassistant", mux)
}

type handler struct{}

func (h *handler) Homes(w http.ResponseWriter, req *http.Request) {
	mu.Lock()
	defer mu.Unlock()

	jsonWrite(w, slices.Sorted(maps.Keys(instances)))
}

func (h *handler) Home(w http.ResponseWriter, req *http.Request) {
	home := req.PathValue("home")
	if instanceByName(home) == nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	conn, _ := NewConnection(log, home)
	res, err := conn.GetStates()
	if err != nil {
		jsonError(w, http.StatusBadRequest, err.Error())
		return
	}

	// cache list of entities
	w.Header().Set("Cache-control", "max-age=300")

	jsonWrite(w, lo.Map(res, func(e StateResponse, _ int) string {
		return e.EntityId
	}))
}

// jsonWrite writes a JSON response
func jsonWrite(w http.ResponseWriter, data any) {
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.ERROR.Printf("homeassistant: failed to encode JSON: %v", err)
	}
}

type errorResponse struct {
	Error string `json:"error"`
}

// jsonError writes an error response
func jsonError(w http.ResponseWriter, status int, message string) {
	w.WriteHeader(status)
	jsonWrite(w, errorResponse{Error: message})
}
