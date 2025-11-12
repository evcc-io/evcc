package homeassistant

import (
	"encoding/json"
	"fmt"
	"maps"
	"net/http"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/server/service"
)

func init() {
	service.Register("homeassistant", new(handler))
}

type handler struct{}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	fmt.Println(req.URL)

	segments := strings.Split(req.URL.Path, "/")
	if len(segments) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")

	switch segments[0] {
	case "homes":
		mu.Lock()
		defer mu.Unlock()

		b, _ := json.Marshal(slices.Sorted(maps.Keys(instances)))
		fmt.Fprint(w, string(b))

	case "home":
		if len(segments) != 3 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		fmt.Fprint(w, `["foo","bar"]`)

	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}
