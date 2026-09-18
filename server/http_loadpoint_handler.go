package server

import (
	"errors"
	"fmt"
	"net/http"
	"slices"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/planner"
	"github.com/evcc-io/evcc/core/site"
	"github.com/gorilla/mux"
	"github.com/samber/lo"
)

type PlanResponse struct {
	PlanId   int       `json:"planId"`
	PlanTime time.Time `json:"planTime"`
	Duration int64     `json:"duration"`
	Plan     api.Rates `json:"plan"`
	Power    float64   `json:"power"`
}

type PlanPreviewResponse struct {
	PlanTime time.Time `json:"planTime"`
	Duration int64     `json:"duration"`
	Plan     api.Rates `json:"plan"`
	Power    float64   `json:"power"`
}

// metaPlanHandler returns the current plan
func metaPlanHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var res struct {
			MetaPlan   PlanPreviewResponse `json:"metaPlan"`
			Loadpoints []PlanResponse      `json:"loadpoints"`
		}

		var totalDuration time.Duration

		for _, lp := range site.Loadpoints() {
			maxPower := lp.EffectiveMaxPower()
			planTime := lp.EffectivePlanTime()
			id := lp.EffectivePlanId()

			goal, _ := lp.GetPlanGoal()
			requiredDuration := lp.GetPlanRequiredDuration(goal, maxPower)

			res.Loadpoints = append(res.Loadpoints, PlanResponse{
				PlanId:   id,
				PlanTime: planTime,
				Duration: int64(requiredDuration.Seconds()),
				Power:    maxPower,
			})

			totalDuration += requiredDuration
		}

		latestEnd := lo.MinBy(res.Loadpoints, func(a, b PlanResponse) bool {
			return a.PlanTime.Before(b.PlanTime)
		}).PlanTime

		for i := 0; i < len(res.Loadpoints); i++ {
			for j := i + 1; j < len(res.Loadpoints); j++ {
				lpi := res.Loadpoints[i]
				lpj := res.Loadpoints[j]

				maxStart := slices.MaxFunc([]time.Time{
					lpi.PlanTime.Add(-time.Duration(lpi.Duration) * time.Second),
					lpj.PlanTime.Add(-time.Duration(lpj.Duration) * time.Second)},
					time.Time.Compare)
				minEnd := slices.MinFunc([]time.Time{lpi.PlanTime, lpj.PlanTime}, time.Time.Compare)

				// add overlapping ranges to total require duration
				if d := minEnd.Sub(maxStart); d > 0 {
					totalDuration += d
				}
			}
		}

		tariff := site.GetTariff(api.TariffUsagePlanner)
		planner := planner.New(util.NewLogger("meta-plan"), tariff)

		metaPlan := planner.Plan(totalDuration, 0, latestEnd)

		res.MetaPlan = PlanPreviewResponse{
			PlanTime: latestEnd,
			Duration: int64(totalDuration.Seconds()),
			Plan:     metaPlan,
		}

		jsonWrite(w, res)
	}
}

// planHandler returns the current plan
func planHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		maxPower := lp.EffectiveMaxPower()
		planTime := lp.EffectivePlanTime()
		id := lp.EffectivePlanId()

		goal, _ := lp.GetPlanGoal()
		requiredDuration := lp.GetPlanRequiredDuration(goal, maxPower)
		strategy := lp.EffectivePlanStrategy()
		plan := lp.GetPlan(planTime, requiredDuration, strategy.Precondition, strategy.Continuous)

		res := PlanResponse{
			PlanId:   id,
			PlanTime: planTime,
			Duration: int64(requiredDuration.Seconds()),
			Plan:     plan,
			Power:    maxPower,
		}

		jsonWrite(w, res)
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
		strategy := lp.EffectivePlanStrategy()

		plan := lp.GetPlan(planTime, requiredDuration, strategy.Precondition, strategy.Continuous)

		res := PlanPreviewResponse{
			PlanTime: planTime,
			Duration: int64(requiredDuration.Seconds()),
			Plan:     plan,
			Power:    maxPower,
		}

		jsonWrite(w, res)
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

		jsonWrite(w, res)
	}
}

// planRemoveHandler removes plan time
func planRemoveHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := lp.SetPlanEnergy(time.Time{}, 0); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, struct{}{})
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
		jsonWrite(w, struct{}{})
	}
}

// vehicleDetectHandler starts vehicle detection
func vehicleDetectHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lp.StartVehicleDetection()
		jsonWrite(w, struct{}{})
	}
}

// planStrategyHandler updates plan strategy for loadpoint
func planStrategyHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := planStrategyHandlerSetter(r, lp.SetPlanStrategy); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := lp.GetPlanStrategy()

		jsonWrite(w, res)
	}
}
