package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/gorilla/mux"
)

// planSocHandler updates plan soc and time
func planSocHandler(v vehicle.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		timeS, ok := vars["time"]
		timeV, err := time.Parse(time.RFC3339, timeS)

		if !ok || err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		valueS, ok := vars["value"]
		valueV, err := strconv.ParseFloat(valueS, 64)

		if !ok || err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := v.SetPlanSoc(timeV, valueV); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := struct {
			Soc  float64   `json:"soc"`
			Time time.Time `json:"time"`
		}{
			Soc:  v.GetPlanSoc(),
			Time: v.GetPlanTime(),
		}

		jsonResult(w, res)
	}
}

// planSocRemoveHandler removes plan soc and time
func planSocRemoveHandler(v vehicle.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := v.SetPlanSoc(time.Time{}, 0); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := struct{}{}
		jsonResult(w, res)
	}
}
