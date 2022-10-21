package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"strconv"
	"text/template"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/db"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	dbserver "github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/locale"
	"github.com/gorilla/mux"
)

func indexHandler() http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		indexTemplate, err := fs.ReadFile(Assets, "index.html")
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

func csvResult(ctx context.Context, w http.ResponseWriter, res any) {
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", `attachment; filename="sessions.csv"`)

	if ww, ok := res.(api.CsvWriter); ok {
		ww.WriteCsv(ctx, w)
	} else {
		w.WriteHeader(http.StatusInternalServerError)
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

// stateHandler returns current charge mode
func stateHandler(cache *util.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := cache.State()
		for _, k := range []string{"availableVersion", "releaseNotes"} {
			delete(res, k)
		}
		jsonResult(w, res)
	}
}

// sessionHandler returns the list of charging sessions
func sessionHandler(w http.ResponseWriter, r *http.Request) {
	if dbserver.Instance == nil {
		jsonError(w, http.StatusBadRequest, errors.New("database offline"))
		return
	}

	var res db.Sessions
	if txn := dbserver.Instance.Where("charged_kwh>=0.05").Order("created desc").Find(&res); txn.Error != nil {
		jsonError(w, http.StatusInternalServerError, txn.Error)
		return
	}

	if r.URL.Query().Get("format") == "csv" {
		accept := r.Header.Get("Accept-Language")
		ctx := context.WithValue(context.Background(), locale.Locale, accept)
		csvResult(ctx, w, &res)
		return
	}

	jsonResult(w, res)
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

// targetChargeHandler updates target soc
func targetChargeHandler(loadpoint targetCharger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		socS, ok := vars["soc"]
		socV, err := strconv.Atoi(socS)

		if !ok || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		timeS, ok := vars["time"]
		timeV, err := time.Parse(time.RFC3339, timeS)

		if !ok || err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		loadpoint.SetTargetCharge(timeV, socV)

		res := struct {
			SoC  int       `json:"soc"`
			Time time.Time `json:"time"`
		}{
			SoC:  socV,
			Time: timeV,
		}

		jsonResult(w, res)
	}
}

// targetChargeRemoveHandler removes target soc
func targetChargeRemoveHandler(loadpoint loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadpoint.SetTargetCharge(time.Time{}, 0)
		res := struct{}{}
		jsonResult(w, res)
	}
}

// vehicleHandler sets active vehicle
func vehicleHandler(site site.API, loadpoint loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		valS, ok := vars["vehicle"]
		val, err := strconv.Atoi(valS)

		vehicles := site.GetVehicles()
		if !ok || val >= len(vehicles) || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		loadpoint.SetVehicle(vehicles[val])

		res := struct {
			Vehicle string `json:"vehicle"`
		}{
			Vehicle: vehicles[val].Title(),
		}

		jsonResult(w, res)
	}
}

// vehicleRemoveHandler removes vehicle
func vehicleRemoveHandler(loadpoint loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadpoint.SetVehicle(nil)
		res := struct{}{}
		jsonResult(w, res)
	}
}

// vehicleDetectHandler starts vehicle detection
func vehicleDetectHandler(loadpoint loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadpoint.StartVehicleDetection()
		res := struct{}{}
		jsonResult(w, res)
	}
}

// socketHandler attaches websocket handler to uri
func socketHandler(hub *SocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ServeWebsocket(hub, w, r)
	}
}

// TargetCharger defines target charge related loadpoint operations
type targetCharger interface {
	// SetTargetCharge sets the charge targetSoC
	SetTargetCharge(time.Time, int)
}
