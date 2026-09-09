package network

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/db/settings"
	"github.com/evcc-io/evcc/server/service"
)

var config globalconfig.Network

const CallbackPath = "/providerauth/callback"

func init() {
	// auth service is registered here to avoid import cycle
	mux := http.NewServeMux()
	mux.HandleFunc("GET /origin", originHandler(func(origin string) string { return origin }))
	mux.HandleFunc("GET /redirecturi", originHandler(func(origin string) string { return origin + CallbackPath }))

	service.Register("auth", mux)
}

func Start(conf globalconfig.Network) {
	config = conf
}

func Config() globalconfig.Network {
	return config
}

func remoteSettings() (enabled bool, origin string) {
	var remote struct {
		Enabled bool   `json:"enabled"`
		URL     string `json:"url"`
	}
	if err := settings.Json(keys.Remote, &remote); err != nil {
		return false, ""
	}
	return remote.Enabled, strings.TrimRight(remote.URL, "/")
}

// RemoteOrigin returns the remote access url as public https origin of this instance.
// The url is kept while remote access is disabled, so devices set up through it keep working.
func RemoteOrigin() string {
	_, origin := remoteSettings()
	return origin
}

// originHandler serves f(origin). With ?remote the origin is the remote access url, none while disabled.
func originHandler(f func(origin string) string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		origin := config.ExternalURL()
		if req.URL.Query().Has("remote") {
			enabled, remote := remoteSettings()
			origin = ""
			if enabled {
				origin = remote
			}
		}

		res := []string{}
		if origin != "" {
			res = append(res, f(origin))
		}
		json.NewEncoder(w).Encode(res)
	}
}
