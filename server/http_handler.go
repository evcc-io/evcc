package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"math"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/jq"
	"github.com/gorilla/mux"
	"github.com/itchyny/gojq"
)

var ignoreState = []string{"releaseNotes"} // excessive size

func indexHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		indexTemplate, err := fs.ReadFile(assets.Web, "index.html")
		if err != nil {
			log.FATAL.Print("httpd: failed to load embedded template:", err.Error())
			log.FATAL.Print("Make sure templates are included using the `release` build tag or use `make build`")
			w.WriteHeader(http.StatusNotFound)
			return
		}

		t, err := template.New("evcc").Delims("[[", "]]").Parse(string(indexTemplate))
		if err != nil {
			log.FATAL.Fatal("httpd: failed to create main page template:", err.Error())
		}

		if err := t.Execute(w, map[string]interface{}{
			"Version": Version,
			"Commit":  Commit,
		}); err != nil {
			log.ERROR.Println("httpd: failed to render main page:", err.Error())
		}
	})
}

// jsonHandler is a middleware that decorates responses with JSON and CORS headers
func jsonHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=UTF-8")
		h.ServeHTTP(w, r)
	})
}

func jsonWrite(w http.ResponseWriter, content interface{}) {
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.ERROR.Printf("httpd: failed to encode JSON: %v", err)
	}
}

func jsonResult(w http.ResponseWriter, res interface{}) {
	jsonWrite(w, map[string]interface{}{"result": res})
}

func jsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	jsonWrite(w, map[string]interface{}{"error": err.Error()})
}

// pass converts a simple api without return value to api with nil error return value
func pass[T any](f func(T)) func(T) error {
	return func(v T) error {
		f(v)
		return nil
	}
}

// floatHandler updates float-param api
func floatHandler(set func(float64) error, get func() float64) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		val, err := strconv.ParseFloat(vars["value"], 64)
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

// intHandler updates int-param api
func intHandler(set func(int) error, get func() int) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		val, err := strconv.Atoi(vars["value"])
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

// boolHandler updates bool-param api
func boolHandler(set func(bool) error, get func() bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		val, err := strconv.ParseBool(vars["value"])
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

// boolGetHandler retrievs bool api values
func boolGetHandler(get func() bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		jsonResult(w, get())
	}
}

// encodeFloats replaces NaN and Inf with nil
// TODO handle hierarchical data
func encodeFloats(data map[string]any) {
	for k, v := range data {
		switch v := v.(type) {
		case float64:
			if math.IsNaN(v) || math.IsInf(v, 0) {
				data[k] = nil
			}
		}
	}
}

// stateHandler returns the combined state
func stateHandler(cache *util.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := cache.State()
		for _, k := range ignoreState {
			delete(res, k)
		}

		encodeFloats(res)

		if q := r.URL.Query().Get("jq"); q != "" {
			q = strings.TrimPrefix(q, ".result")

			query, err := gojq.Parse(q)
			if err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			b, err := json.Marshal(res)
			if err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			res, err := jq.Query(query, b)
			if err != nil {
				jsonError(w, http.StatusBadRequest, err)
				return
			}

			jsonWrite(w, res)
			return
		}

		jsonResult(w, res)
	}
}

// healthHandler returns current charge mode
func healthHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if site == nil || !site.Healthy() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
	}
}

// tariffHandler returns the configured tariff
func tariffHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		tariff := vars["tariff"]

		t := site.GetTariff(tariff)
		if t == nil {
			jsonError(w, http.StatusNotFound, errors.New("tariff not available"))
			return
		}

		rates, err := t.Rates()
		if err != nil {
			jsonError(w, http.StatusNotFound, err)
			return
		}

		res := struct {
			Rates api.Rates `json:"rates"`
		}{
			Rates: rates,
		}

		jsonResult(w, res)
	}
}

// chargeModeHandler updates charge mode
func chargeModeHandler(lp loadpoint.API) http.HandlerFunc {
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

// phasesHandler updates minimum soc
func phasesHandler(lp loadpoint.API) http.HandlerFunc {
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

// targetTimeHandler updates target soc
func targetTimeHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		timeS, ok := vars["time"]
		timeV, err := time.Parse(time.RFC3339, timeS)

		if !ok || err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		if err := lp.SetTargetTime(timeV); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := struct {
			Soc    int       `json:"soc"`
			Energy float64   `json:"energy"`
			Time   time.Time `json:"time"`
		}{
			Soc:    lp.GetTargetSoc(),
			Energy: lp.GetTargetEnergy(),
			Time:   lp.GetTargetTime(),
		}

		jsonResult(w, res)
	}
}

// targetTimeRemoveHandler removes target soc
func targetTimeRemoveHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := lp.SetTargetTime(time.Time{}); err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		res := struct{}{}
		jsonResult(w, res)
	}
}

// vehicleHandler sets active vehicle
func vehicleHandler(site site.API, lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		valS, ok := vars["vehicle"]
		val, err := strconv.Atoi(valS)

		vehicles := site.GetVehicles()
		if !ok || val < 1 || val > len(vehicles) || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		v := vehicles[val-1]
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

// planHandler starts vehicle detection
func planHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var err error

		targetTime := lp.GetTargetTime()
		if t := r.URL.Query().Get("targetTime"); t != "" {
			targetTime, err = time.Parse(time.RFC3339, t)
		}

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		power := lp.GetMaxPower()
		requiredDuration, plan, err := lp.GetPlan(targetTime, power)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		res := struct {
			Duration int64     `json:"duration"`
			Plan     api.Rates `json:"plan"`
			Unit     string    `json:"unit"`
			Power    float64   `json:"power"`
		}{
			Duration: int64(requiredDuration.Seconds()),
			Plan:     plan,
			Power:    power,
		}
		jsonResult(w, res)
	}
}

// socketHandler attaches websocket handler to uri
func socketHandler(hub *SocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		hub.ServeWebsocket(w, r)
	}
}
