package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/gorilla/mux"
)

type moder interface {
	GetMode() api.ChargeMode
	SetMode(api.ChargeMode)
}

// chargeModeHandler updates charge mode
func chargeModeHandler(lp moder) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		mode, err := api.ChargeModeString(vars["value"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		lp.SetMode(mode)

		jsonResult(w, lp.GetMode())
	}
}

type phaser interface {
	GetPhases() int
	SetPhases(int) error
}

// phasesHandler updates minimum soc
func phasesHandler(lp phaser) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		phases, err := strconv.Atoi(vars["value"])
		if err == nil {
			err = lp.SetPhases(phases)
		}

		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonResult(w, lp.GetPhases())
	}
}

// remoteDemandHandler updates minimum soc
func remoteDemandHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		source := vars["source"]
		demand, err := loadpoint.RemoteDemandString(vars["demand"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		lp.RemoteControl(source, demand)

		res := struct {
			Demand loadpoint.RemoteDemand `json:"demand"`
			Source string                 `json:"source"`
		}{
			Source: source,
			Demand: demand,
		}

		jsonResult(w, res)
	}
}

// planHandler returns the current effective plan
func planHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		planTime := lp.EffectivePlanTime()
		power := lp.EffectiveMaxPower()
		requiredDuration, plan, err := lp.GetPlan(planTime, power)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		res := struct {
			Duration int64     `json:"duration"`
			Plan     api.Rates `json:"plan"`
			Power    float64   `json:"power"`
		}{
			Duration: int64(requiredDuration.Seconds()),
			Plan:     plan,
			Power:    power,
		}

		jsonResult(w, res)
	}
}

// planEnergyHandler updates plan energy and time
func planEnergyHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		ts, err := time.Parse(time.RFC3339, vars["time"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		val, err := strconv.ParseFloat(vars["value"], 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := lp.SetPlanEnergy(ts, val); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		ts, energy := lp.GetPlanEnergy()

		res := struct {
			Energy float64   `json:"energy"`
			Time   time.Time `json:"time"`
		}{
			Energy: energy,
			Time:   ts,
		}

		jsonResult(w, res)
	}
}

// planRemoveHandler removes plan time
func planRemoveHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := lp.SetPlanEnergy(time.Time{}, 0); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := struct{}{}
		jsonResult(w, res)
	}
}

// vehicleSelectHandler sets active vehicle
func vehicleSelectHandler(site site.API, lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		vv, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		v := vv.Instance()
		lp.SetVehicle(v)

		res := struct {
			Vehicle string `json:"vehicle"`
		}{
			Vehicle: v.Title(),
		}

		jsonResult(w, res)
	}
}

// vehicleRemoveHandler removes vehicle
func vehicleRemoveHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lp.SetVehicle(nil)
		res := struct{}{}
		jsonResult(w, res)
	}
}

// vehicleDetectHandler starts vehicle detection
func vehicleDetectHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lp.StartVehicleDetection()
		res := struct{}{}
		jsonResult(w, res)
	}
}
