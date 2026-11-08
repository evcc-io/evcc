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

// httpsRoot returns scheme and host of an https url served at the domain root, else empty
func httpsRoot(s string) string {
	u, err := url.Parse(s)
	if err != nil || u.Scheme != "https" || u.Host == "" || strings.Trim(u.Path, "/") != "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

// PublicURL returns the public https url of this instance (scheme and host only): the enabled remote access url, else an https external url.
func PublicURL() string {
	var remote struct {
		Enabled bool   `json:"enabled"`
		URL     string `json:"url"`
	}
	if err := settings.Json(keys.Remote, &remote); err == nil && remote.Enabled {
		return httpsRoot(remote.URL)
	}
	return httpsRoot(config.ExternalUrl)
}

// originHandler serves f(origin). With ?public the origin is the public url, none if unavailable.
func originHandler(f func(origin string) string) http.HandlerFunc {
	return func(w http.ResponseWriter, req *http.Request) {
		origin := config.ExternalURL()
		if req.URL.Query().Has("public") {
			origin = PublicURL()
		}

		res := []string{}
		if origin != "" {
			res = append(res, f(origin))
		}
		json.NewEncoder(w).Encode(res)
	}
}
