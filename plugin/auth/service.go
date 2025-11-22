package auth

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/server/service"
)

func init() {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /redirecturi", getRedirectUri)
	service.Register("auth", mux)
}

func getRedirectUri(w http.ResponseWriter, req *http.Request) {
	uri := network.Config().ExternalUrl + "/providerauth/callback"
	json.NewEncoder(w).Encode([]string{uri})
}
