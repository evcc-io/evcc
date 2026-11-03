package server

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/core/vehicle"
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

// vehicleModeHandler updates the vehicle charge mode (empty value clears it)
func vehicleModeHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		mode, err := api.ChargeModeString(vars["value"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		v.SetMode(mode)

		res := struct {
			Mode api.ChargeMode `json:"mode"`
		}{
			Mode: v.GetMode(),
		}

		jsonWrite(w, res)
	}
}

// vehicleAlwaysChargeHandler updates the vehicle always charge default (empty value clears it)
func vehicleAlwaysChargeHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		// route restricts value to on|off, DELETE has none
		v.SetAlwaysCharge(api.AlwaysCharge(vars["value"]))

		res := struct {
			AlwaysCharge api.AlwaysCharge `json:"alwaysCharge"`
		}{
			AlwaysCharge: v.GetAlwaysCharge(),
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

		site.Optimize()

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

func planStrategyHandlerSetter(r *http.Request, set func(api.PlanStrategy) error) error {
	var res api.PlanStrategy
	if err := json.NewDecoder(r.Body).Decode(&res); err != nil {
		return err
	}

	return set(res)
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

		if err := planStrategyHandlerSetter(r, v.SetPlanStrategy); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := v.GetPlanStrategy()

		jsonWrite(w, res)
	}
}

// fullChargeHandler updates the periodic full charge thresholds
func fullChargeHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		b, err := io.ReadAll(r.Body)
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := updateFullCharge(v, b); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonWrite(w, v.GetFullCharge())
	}
}

// updateFullCharge changes the thresholds present in the payload and keeps the others
func updateFullCharge(v vehicle.API, payload []byte) error {
	var req struct {
		Days   *int     `json:"days"`
		Energy *float64 `json:"energy"`
	}
	if err := json.Unmarshal(payload, &req); err != nil {
		return err
	}

	res := v.GetFullCharge()
	if req.Days != nil {
		res.Days = *req.Days
	}
	if req.Energy != nil {
		res.Energy = *req.Energy
	}

	return v.SetFullCharge(res)
}

// fullChargeDone records a full charge that happened outside of evcc, e.g. at a public charger
func fullChargeDone(v vehicle.API, ts time.Time) error {
	if ts.After(time.Now()) {
		return errors.New("timestamp is in the future")
	}

	v.SetFullChargeTime(ts)

	return nil
}

// fullChargeDoneHandler records a full charge, now or at the given time
func fullChargeDoneHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		v, err := site.Vehicles().ByName(vars["name"])
		if err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		ts := time.Now()
		if s, ok := vars["time"]; ok {
			if ts, err = time.Parse(time.RFC3339, s); err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}
		}

		if err := fullChargeDone(v, ts); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := struct {
			Time time.Time `json:"time"`
		}{
			Time: v.GetFullChargeTime(),
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

		site.Optimize()

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

		if err := v.SetPlanSoc(time.Time{}, 0); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		site.Optimize()

		jsonWrite(w, struct{}{})
	}
}
