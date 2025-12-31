package server

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/gorilla/mux"
)

// minSocHandler updates min soc
func minSocHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
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

		jsonWrite(w, res)
	}
}

// limitSocHandler updates limit soc
func limitSocHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
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

		jsonWrite(w, res)
	}
}

// planSocHandler updates plan soc and time
func planSocHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		query := r.URL.Query()

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		ts, err := time.ParseInLocation(time.RFC3339, vars["time"], nil)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		soc, err := strconv.Atoi(vars["value"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		precondition, err := parseDuration(query.Get("precondition"))
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := v.SetPlanSoc(ts, precondition, soc); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		ts, precondition, soc = v.GetPlanSoc()

		res := struct {
			Soc          int       `json:"soc"`
			Precondition int64     `json:"precondition"`
			Time         time.Time `json:"time"`
		}{
			Soc:          soc,
			Precondition: int64(precondition.Seconds()),
			Time:         ts,
		}

		jsonWrite(w, res)
	}
}

// addRepeatingPlansHandler handles any information regarding weekday, hour, minute, soc and isActive
func addRepeatingPlansHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		var res []api.RepeatingPlan
		if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := v.SetRepeatingPlans(res); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, res)
	}
}

// planSocRemoveHandler removes plan soc and time
func planSocRemoveHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := v.SetPlanSoc(time.Time{}, 0, 0); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := struct{}{}
		jsonWrite(w, res)
	}
}
