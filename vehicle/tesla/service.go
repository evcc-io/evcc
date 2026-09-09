package tesla

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/server/service"
)

func init() {
	mux := http.NewServeMux()

	// virtual key link the user opens on the phone, derived from the domain registered with tesla
	mux.HandleFunc("GET /virtualkey", func(w http.ResponseWriter, req *http.Request) {
		res := []string{}
		if host, err := originHost(network.RemoteOrigin()); err == nil {
			res = append(res, "https://tesla.com/_ak/"+host)
		}
		_ = json.NewEncoder(w).Encode(res)
	})

	service.Register("tesla", mux)
}
