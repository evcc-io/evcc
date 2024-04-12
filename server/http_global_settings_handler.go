package server

import (
	"net/http"
	"strings"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/util"
	"gopkg.in/yaml.v3"
)

func settingsGetStringHandler(key string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res, _ := settings.String(key)
		jsonResult(w, res)
	}
}

func settingsSetYamlHandler(key string, struc any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		other := make(map[string]interface{})
		if err := yaml.NewDecoder(r.Body).Decode(&other); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := util.DecodeOther(other, &struc); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		var res strings.Builder
		enc := yaml.NewEncoder(&res)
		enc.SetIndent(2)

		if err := enc.Encode(struc); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		val := res.String()
		settings.SetString(key, val)

		setConfigDirty()

		w.WriteHeader(http.StatusOK)
		jsonResult(w, val)
	}
}

func settingsFloatHandler(key string) http.HandlerFunc {
	return floatHandler(func(val float64) error {
		settings.SetFloat(key, val)
		return nil
	}, func() float64 {
		res, _ := settings.Float(key)
		return res
	})
}
