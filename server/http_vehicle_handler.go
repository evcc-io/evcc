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

		if err := v.SetPlanSoc(ts, soc); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		ts, soc = v.GetPlanSoc()

		res := struct {
			Soc  int       `json:"soc"`
			Time time.Time `json:"time"`
		}{
			Soc:  soc,
			Time: ts,
		}

		jsonWrite(w, res)
	}
}

func planStrategyHandlerSetter(w http.ResponseWriter, r *http.Request, set func(api.PlanStrategy) error) error {
	var planStrategy planStrategyPayload
	if err := json.NewDecoder(r.Body).Decode(&planStrategy); err != nil {
		return err
	}
	return set(api.PlanStrategy{
		Continuous:   planStrategy.Continuous,
		Precondition: time.Duration(planStrategy.Precondition) * time.Second,
	})
}

// updatePlanStrategyHandler updates plan strategy
func updatePlanStrategyHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := planStrategyHandlerSetter(w, r, v.SetPlanStrategy); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := planStrategyPayloadFromApi(v.GetPlanStrategy())

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

		var plansWrapper struct {
			RepeatingPlans []api.RepeatingPlanStruct `json:"plans"`
		}

		if err := json.NewDecoder(r.Body).Decode(&plansWrapper); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := v.SetRepeatingPlans(plansWrapper.RepeatingPlans); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, plansWrapper)
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

		if err := v.SetPlanSoc(time.Time{}, 0); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, struct{}{})
	}
}
