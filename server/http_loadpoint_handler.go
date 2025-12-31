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

type PlanResponse struct {
	PlanId       int       `json:"planId"`
	PlanTime     time.Time `json:"planTime"`
	Duration     int64     `json:"duration"`
	Precondition int64     `json:"precondition"`
	Plan         api.Rates `json:"plan"`
	Power        float64   `json:"power"`
}

type PlanPreviewResponse struct {
	PlanTime     time.Time `json:"planTime"`
	Duration     int64     `json:"duration"`
	Precondition int64     `json:"precondition"`
	Plan         api.Rates `json:"plan"`
	Power        float64   `json:"power"`
}

// planHandler returns the current plan
func planHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		maxPower := lp.EffectiveMaxPower()
		planTime := lp.EffectivePlanTime()
		id := lp.EffectivePlanId()

		goal, _ := lp.GetPlanGoal()
		precondition := lp.GetPlanPreCondDuration()
		requiredDuration := lp.GetPlanRequiredDuration(goal, maxPower)
		plan := lp.GetPlan(planTime, requiredDuration, precondition)

		res := PlanResponse{
			PlanId:       id,
			PlanTime:     planTime,
			Duration:     int64(requiredDuration.Seconds()),
			Precondition: int64(precondition.Seconds()),
			Plan:         plan,
			Power:        maxPower,
		}

		jsonWrite(w, res)
	}
}

// staticPlanPreviewHandler returns a plan preview for given parameters
func staticPlanPreviewHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		query := r.URL.Query()

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

		precondition, err := parseDuration(query.Get("precondition"))
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
		plan := lp.GetPlan(planTime, requiredDuration, precondition)

		res := PlanPreviewResponse{
			PlanTime:     planTime,
			Duration:     int64(requiredDuration.Seconds()),
			Precondition: int64(precondition.Seconds()),
			Plan:         plan,
			Power:        maxPower,
		}

		jsonWrite(w, res)
	}
}

func repeatingPlanPreviewHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		query := r.URL.Query()

		hourMinute := vars["time"]
		tz := vars["tz"]

		var weekdays []int
		for weekdayStr := range strings.SplitSeq(vars["weekdays"], ",") {
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

		precondition, err := parseDuration(query.Get("precondition"))
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		maxPower := lp.EffectiveMaxPower()
		requiredDuration := lp.GetPlanRequiredDuration(soc, maxPower)
		plan := lp.GetPlan(planTime, requiredDuration, precondition)

		res := PlanPreviewResponse{
			PlanTime:     planTime,
			Duration:     int64(requiredDuration.Seconds()),
			Precondition: int64(precondition.Seconds()),
			Plan:         plan,
			Power:        maxPower,
		}

		jsonWrite(w, res)
	}
}

// planEnergyHandler updates plan energy and time
func planEnergyHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		query := r.URL.Query()

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

		precondition, err := parseDuration(query.Get("precondition"))
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := lp.SetPlanEnergy(ts, precondition, val); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		ts, precondition, energy := lp.GetPlanEnergy()

		res := struct {
			Energy       float64   `json:"energy"`
			Precondition int64     `json:"precondition"`
			Time         time.Time `json:"time"`
		}{
			Energy:       energy,
			Precondition: int64(precondition.Seconds()),
			Time:         ts,
		}

		jsonWrite(w, res)
	}
}

// planRemoveHandler removes plan time
func planRemoveHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := lp.SetPlanEnergy(time.Time{}, 0, 0); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := struct{}{}
		jsonWrite(w, res)
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
			Vehicle: v.GetTitle(),
		}

		jsonWrite(w, res)
	}
}

// vehicleRemoveHandler removes vehicle
func vehicleRemoveHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lp.SetVehicle(nil)
		res := struct{}{}
		jsonWrite(w, res)
	}
}

// vehicleDetectHandler starts vehicle detection
func vehicleDetectHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lp.StartVehicleDetection()
		res := struct{}{}
		jsonWrite(w, res)
	}
}
