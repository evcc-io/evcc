package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
)

func solarMinSocConfigHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			jsonWrite(w, site.GetSolarMinSoc())
			return
		}

		var conf api.SolarMinSocConfig
		if err := json.NewDecoder(r.Body).Decode(&conf); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}
		if err := site.SetSolarMinSoc(conf); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, site.GetSolarMinSoc())
	}
}
