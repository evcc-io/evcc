package server

import (
	"encoding/json"
	"net/http"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util/config"
)

// tariffsHandler returns a device configurations by class
func tariffsHandler(site site.API) http.HandlerFunc {
	tariffs := site.GetTariffs()

	return func(w http.ResponseWriter, r *http.Request) {
		res := struct {
			Currency string `json:"currency,omitempty"`
			Grid     string `json:"grid,omitempty"`
			Feedin   string `json:"feedin,omitempty"`
			Co2      string `json:"co2,omitempty"`
			Planner  string `json:"planner,omitempty"`
		}{
			Currency: tariffs.GetCurrency().String(),
			Grid:     tariffs.GetRef(tariff.Grid),
			Feedin:   tariffs.GetRef(tariff.Feedin),
			Co2:      tariffs.GetRef(tariff.Co2),
			Planner:  tariffs.GetRef(tariff.Planner),
		}

		jsonResult(w, res)
	}
}

// updateTariffsHandler returns a device configurations by class
func updateTariffsHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload struct {
			Currency *string `json:"currency"`
			Grid     *string `json:"grid"`
			Feedin   *string `json:"feedin"`
			Co2      *string `json:"co2"`
			Planner  *string `json:"planner"`
		}

		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		tariffs := site.GetTariffs()

		if payload.Currency != nil {
			if err := tariffs.SetCurrency(*payload.Currency); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			setConfigDirty()
		}

		if payload.Grid != nil {
			ref := *payload.Grid
			if ref != "" && !validateRefs(w, config.Tariffs(), []string{ref}) {
				return
			}

			tariffs.SetRef(tariff.Grid, ref)
			setConfigDirty()
		}

		if payload.Feedin != nil {
			ref := *payload.Feedin
			if ref != "" && !validateRefs(w, config.Tariffs(), []string{ref}) {
				return
			}

			tariffs.SetRef(tariff.Feedin, ref)
			setConfigDirty()
		}

		if payload.Co2 != nil {
			ref := *payload.Co2
			if ref != "" && !validateRefs(w, config.Tariffs(), []string{ref}) {
				return
			}

			tariffs.SetRef(tariff.Co2, ref)
			setConfigDirty()
		}

		if payload.Planner != nil {
			ref := *payload.Planner
			if ref != "" && !validateRefs(w, config.Tariffs(), []string{ref}) {
				return
			}

			tariffs.SetRef(tariff.Planner, ref)
			setConfigDirty()
		}

		status := map[bool]int{false: http.StatusOK, true: http.StatusAccepted}
		w.WriteHeader(status[ConfigDirty()])
	}
}
