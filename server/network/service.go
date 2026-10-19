package network

import (
	"encoding/json"
	"net/http"
	"net/url"
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

// httpsExternalURL returns the external url if explicitly configured as https, e.g. behind a reverse proxy
func httpsExternalURL() string {
	if u, err := url.Parse(config.ExternalUrl); err != nil || u.Scheme != "https" || u.Host == "" {
		return ""
	}
	return strings.TrimRight(config.ExternalUrl, "/")
}

// PublicOrigin returns the public https origin of this instance: the remote access url, else an https external url.
// The remote access url is kept while remote access is disabled, so devices set up through it keep working.
func PublicOrigin() string {
	enabled, remote := remoteSettings()
	if enabled {
		return remote
	}
	if origin := httpsExternalURL(); origin != "" {
		return origin
	}
	return remote
}

// originHandler serves f(origin). With ?public the origin is the public origin for new setups, none if unavailable.
func originHandler(f func(origin string) string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		origin := config.ExternalURL()
		if req.URL.Query().Has("public") {
			origin = httpsExternalURL()
			if enabled, remote := remoteSettings(); enabled {
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
