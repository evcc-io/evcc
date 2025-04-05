package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/mux"
)

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

// planHandler returns the current plan
func planHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		maxPower := lp.EffectiveMaxPower()
		planTime := lp.EffectivePlanTime()
		id := lp.EffectivePlanId()

		// TODO late option
		goal, _ := lp.GetPlanGoal()
		requiredDuration := lp.GetPlanRequiredDuration(goal, maxPower)
		plan := lp.GetPlan(planTime, requiredDuration, false)

		res := struct {
			PlanId   int       `json:"planId"`
			PlanTime time.Time `json:"planTime"`
			Duration int64     `json:"duration"`
			Plan     api.Rates `json:"plan"`
			Power    float64   `json:"power"`
		}{
			PlanId:   id,
			PlanTime: planTime,
			Duration: int64(requiredDuration.Seconds()),
			Plan:     plan,
			Power:    maxPower,
		}

		jsonResult(w, res)
	}
}

// staticPlanPreviewHandler returns a plan preview for given parameters
func staticPlanPreviewHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		planTime, err := time.ParseInLocation(time.RFC3339, vars["time"], nil)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		goal, err := strconv.ParseFloat(vars["value"], 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		late, err := strconv.ParseBool(vars["late"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		switch typ := vars["type"]; typ {
		case "soc":
			if !lp.SocBasedPlanning() {
				jsonError(w, http.StatusBadRequest, errors.New("soc planning only available for vehicles with known soc and capacity"))
				return
			}
		case "energy":
			if lp.SocBasedPlanning() {
				jsonError(w, http.StatusBadRequest, errors.New("energy planning not available for vehicles with known soc and capacity"))
				return
			}
		default:
			jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid plan type: %s", typ))
			return
		}

		maxPower := lp.EffectiveMaxPower()
		requiredDuration := lp.GetPlanRequiredDuration(goal, maxPower)
		plan := lp.GetPlan(planTime, requiredDuration, late)

		res := struct {
			PlanTime time.Time `json:"planTime"`
			Duration int64     `json:"duration"`
			Plan     api.Rates `json:"plan"`
			Power    float64   `json:"power"`
		}{
			PlanTime: planTime,
			Duration: int64(requiredDuration.Seconds()),
			Plan:     plan,
			Power:    maxPower,
		}

		jsonResult(w, res)
	}
}

func repeatingPlanPreviewHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		hourMinute := vars["time"]
		tz := vars["tz"]

		var weekdays []int
		for _, weekdayStr := range strings.Split(vars["weekdays"], ",") {
			weekday, err := strconv.Atoi(weekdayStr)
			if err != nil {
				jsonError(w, http.StatusBadRequest, fmt.Errorf("invalid weekdays format"))
				return
			}
			weekdays = append(weekdays, weekday)
		}

		soc, err := strconv.ParseFloat(vars["soc"], 64)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		planTime, err := util.GetNextOccurrence(weekdays, hourMinute, tz)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// TODO late option
		maxPower := lp.EffectiveMaxPower()
		requiredDuration := lp.GetPlanRequiredDuration(soc, maxPower)
		plan := lp.GetPlan(planTime, requiredDuration, false)

		res := struct {
			PlanTime time.Time `json:"planTime"`
			Duration int64     `json:"duration"`
			Plan     api.Rates `json:"plan"`
			Power    float64   `json:"power"`
		}{
			PlanTime: planTime,
			Duration: int64(requiredDuration.Seconds()),
			Plan:     plan,
			Power:    maxPower,
		}

		jsonResult(w, res)
	}
}

// planEnergyHandler updates plan energy and time
func planEnergyHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		ts, err := time.ParseInLocation(time.RFC3339, vars["time"], nil)
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
