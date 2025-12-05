package network

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

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
	uri := config.ExternalUrl

	if uri == "" {
		// referer
		if referer, err := url.Parse(req.Header.Get("Referer")); err == nil && referer.Host != "" && referer.Scheme != "" {
			hostPort := strings.Split(referer.Host, ":")

			if len(hostPort) > 0 && hostPort[0] != "localhost" && hostPort[0] != "127.0.0.1" {
				uri = referer.Scheme + "://" + referer.Host
			}
		}
	}

	if uri == "" {
		// external url with internal fallback
		uri = config.ExternalURL()
	}

	uri = strings.TrimRight(uri, "/") + CallbackPath
	json.NewEncoder(w).Encode([]string{uri})
}
