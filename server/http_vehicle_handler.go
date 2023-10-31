package server

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/gorilla/mux"
)

func vehicleFromRequest(r *http.Request, site site.API) (vehicle.API, error) {
	name, ok := mux.Vars(r)["name"]
	if !ok {
		return nil, errors.New("invalid name")
	}

	return site.Vehicles().ByName(name)
}

// minSocHandler updates min soc
func minSocHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := vehicleFromRequest(r, site)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		soc, err := strconv.Atoi(vars["value"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		v.SetMinSoc(soc)

		res := struct {
			Soc int `json:"soc"`
		}{
			Soc: v.GetMinSoc(),
		}

		jsonResult(w, res)
	}
}

// limitSocHandler updates limit soc
func limitSocHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := vehicleFromRequest(r, site)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		soc, err := strconv.Atoi(vars["value"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		v.SetLimitSoc(soc)

		res := struct {
			Soc int `json:"soc"`
		}{
			Soc: v.GetLimitSoc(),
		}

		jsonResult(w, res)
	}
}

// planSocHandler updates plan soc and time
func planSocHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := vehicleFromRequest(r, site)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		ts, err := time.Parse(time.RFC3339, vars["time"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		soc, err := strconv.Atoi(vars["value"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := v.SetPlanSoc(ts, soc); err != nil {
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
		v, err := vehicleFromRequest(r, site)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
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
