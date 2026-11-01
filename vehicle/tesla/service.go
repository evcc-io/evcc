package tesla

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/server/service"
)

func init() {
	mux := http.NewServeMux()

	// virtual key link the user opens on the phone, derived from the domain registered with tesla
	mux.HandleFunc("GET /virtualkey", func(w http.ResponseWriter, req *http.Request) {
		res := []string{}
		if origin := network.PublicURL(); origin != "" {
			res = append(res, "https://tesla.com/_ak/"+strings.TrimPrefix(origin, "https://"))
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	service.Register("tesla", mux)
}
