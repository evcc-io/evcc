package tesla

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/server/network"
	"github.com/evcc-io/evcc/server/service"
)

func init() {
	mux := http.NewServeMux()

	// values the user copies into the Tesla developer app, all derived from the remote access url
	for path, f := range map[string]func(origin string) string{
		"/origin":      func(origin string) string { return origin },
		"/redirecturi": func(origin string) string { return origin + network.CallbackPath },
		"/virtualkey": func(origin string) string {
			host, _ := originHost(origin)
			return "https://tesla.com/_ak/" + host
		},
	} {
		mux.HandleFunc("GET "+path, func(w http.ResponseWriter, req *http.Request) {
			res := []string{}
			if origin := remoteOrigin(); origin != "" {
				res = append(res, f(origin))
			}
			_ = json.NewEncoder(w).Encode(res)
		})
	}

	service.Register("tesla", mux)
}

// remoteOrigin returns the remote access url as public origin of this instance, empty if disabled
func remoteOrigin() string {
	var remote struct {
		Enabled bool   `json:"enabled"`
		URL     string `json:"url"`
	}
	if err := settings.Json(keys.Remote, &remote); err != nil || !remote.Enabled {
		return ""
	}
	return strings.TrimRight(remote.URL, "/")
}
