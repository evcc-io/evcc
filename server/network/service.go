package network

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/server/service"
)

var config globalconfig.Network

const CallbackPath = "/providerauth/callback"

func init() {
	// auth service is registered here to avoid import cycle
	mux := http.NewServeMux()
	mux.HandleFunc("GET /redirecturi", getRedirectUri)

	service.Register("auth", mux)
}

func Start(conf globalconfig.Network) {
	config = conf
}

func Config() globalconfig.Network {
	return config
}

func getRedirectUri(w http.ResponseWriter, req *http.Request) {
	uri := config.ExternalURL() + CallbackPath
	json.NewEncoder(w).Encode([]string{uri})
}
