package server

import (
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/core/vehicle"
	"github.com/gorilla/mux"
)

// vehicleHandler updates limit soc
func vehicleHandler[T any](site site.API, h func(v vehicle.API) (func(string) (T, error), func(T) error, func() T)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		conv, set, get := h(v)

		val, err := conv(vars["value"])
		if err == nil {
			err = set(val)
		}

		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonResult(w, get())
	}
}

// vehicleMinSocHandler handles min soc
func vehicleMinSocHandler(v vehicle.API) (func(string) (int, error), func(int) error, func() int) {
	return strconv.Atoi, pass(v.SetMinSoc), v.GetMinSoc
}

// vehicleLimitSocHandler handles limit soc
func vehicleLimitSocHandler(v vehicle.API) (func(string) (int, error), func(int) error, func() int) {
	return strconv.Atoi, pass(v.SetLimitSoc), v.GetLimitSoc
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

		ts, soc = v.GetPlanSoc()

		res := struct {
			Soc  int       `json:"soc"`
			Time time.Time `json:"time"`
		}{
			Soc:  soc,
			Time: ts,
		}

		jsonResult(w, res)
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

		res := struct{}{}
		jsonResult(w, res)
	}
}

// vehicleModeHandler handles mode
func vehicleModeHandler(v vehicle.API) (func(string) (api.ChargeMode, error), func(api.ChargeMode) error, func() api.ChargeMode) {
	return api.ChargeModeString, pass(v.SetMode), v.GetMode
}

// vehiclePhasesHandler handles phases
func vehiclePhasesHandler(v vehicle.API) (func(string) (int, error), func(int) error, func() int) {
	return strconv.Atoi, pass(v.SetPhases), v.GetPhases
}

// vehicleMinCurrentHandler handles min current
func vehicleMinCurrentHandler(v vehicle.API) (func(string) (float64, error), func(float64) error, func() float64) {
	return parseFloat, pass(v.SetMinCurrent), v.GetMinCurrent
}

// vehicleMaxCurrentHandler handles max current
func vehicleMaxCurrentHandler(v vehicle.API) (func(string) (float64, error), func(float64) error, func() float64) {
	return parseFloat, pass(v.SetMaxCurrent), v.GetMaxCurrent
}
