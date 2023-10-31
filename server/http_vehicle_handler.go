package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/core/site"
	"github.com/gorilla/mux"
)

// minSocHandler updates min soc
func minSocHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		name, ok := vars["name"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		_ = name
	}
}

// planSocHandler updates plan soc and time
func planSocHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		timeS, ok := vars["time"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		timeV, err := time.Parse(time.RFC3339, timeS)
		if !ok || err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		valueS, ok := vars["value"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		valueV, err := strconv.Atoi(valueS)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := v.SetPlanSoc(timeV, valueV); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := struct {
			Soc  int       `json:"soc"`
			Time time.Time `json:"time"`
		}{
			Soc:  v.GetPlanSoc(),
			Time: v.GetPlanTime(),
		}

		jsonResult(w, res)
	}
}

// planSocRemoveHandler removes plan soc and time
func planSocRemoveHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		name, ok := vars["name"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := v.SetPlanSoc(time.Time{}, 0); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := struct{}{}
		jsonResult(w, res)
	}
}
