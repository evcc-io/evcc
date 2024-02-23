package server

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/tariff/tariffs"
	"github.com/evcc-io/evcc/util/config"
	"github.com/gorilla/mux"
)

// tariffHandler returns the configured tariff
func tariffHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tariff := vars["tariff"]

		t := site.GetTariff(tariff)
		if t == nil {
			jsonError(w, http.StatusNotFound, errors.New("tariff not available"))
			return
		}

		rates, err := t.Rates()
		if err != nil {
			jsonError(w, http.StatusNotFound, err)
			return
		}

		res := struct {
			Rates api.Rates `json:"rates"`
		}{
			Rates: rates,
		}

		jsonResult(w, res)
	}
}

// tariffsHandler returns a device configurations by class
func tariffsHandler(site site.API) http.HandlerFunc {
	tfs := site.GetTariffs()

	return func(w http.ResponseWriter, r *http.Request) {
		res := struct {
			Currency string `json:"currency,omitempty"`
			Grid     string `json:"grid,omitempty"`
			Feedin   string `json:"feedin,omitempty"`
			Co2      string `json:"co2,omitempty"`
			Planner  string `json:"planner,omitempty"`
		}{
			Currency: tfs.GetCurrency().String(),
			Grid:     tfs.GetRef(tariffs.Grid),
			Feedin:   tfs.GetRef(tariffs.Feedin),
			Co2:      tfs.GetRef(tariffs.Co2),
			Planner:  tfs.GetRef(tariffs.Planner),
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

		tfs := site.GetTariffs()

		if payload.Currency != nil {
			if err := tfs.SetCurrency(*payload.Currency); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			setConfigDirty()
		}

		update := func(tariff string, ref *string) bool {
			if ref != nil {
				if *ref != "" && !validateRefs(w, config.Tariffs(), []string{*ref}) {
					return false
				}

				tfs.SetRef(tariff, *ref)
				setConfigDirty()
			}
			return true
		}

		if !update(tariffs.Grid, payload.Grid) {
			return
		}

		if !update(tariffs.Feedin, payload.Feedin) {
			return
		}

		if !update(tariffs.Co2, payload.Co2) {
			return
		}

		if !update(tariffs.Planner, payload.Planner) {
			return
		}

		status := map[bool]int{false: http.StatusOK, true: http.StatusAccepted}
		w.WriteHeader(status[ConfigDirty()])
	}
}
