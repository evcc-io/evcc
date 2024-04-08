package server

import (
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"gopkg.in/yaml.v3"
)

func settingsGetHandler(key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, _ := settings.String(key)
		jsonResult(w, res)
	}
}

func settingsSetHandler(key string, val any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		other := make(map[string]interface{})
		if err := yaml.NewDecoder(r.Body).Decode(&other); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := util.DecodeOther(other, &val); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		var res strings.Builder
		enc := yaml.NewEncoder(&res)
		enc.SetIndent(2)

		if err := enc.Encode(val); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		settings.SetString(key, res.String())

		w.WriteHeader(http.StatusOK)
		jsonResult(w, res)
	}
}
