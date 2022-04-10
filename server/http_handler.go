package server

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"strconv"
	"text/template"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/log"
	"github.com/gorilla/mux"
)

func indexHandler(site site.API) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		indexTemplate, err := fs.ReadFile(Assets, "index.html")
		if err != nil {
			log.Error("httpd: failed to load embedded template: %s", err.Error())
			log.Error("Make sure templates are included using the `release` build tag or use `make build`")
			os.Exit(1)
		}

		t, err := template.New("evcc").Delims("[[", "]]").Parse(string(indexTemplate))
		if err != nil {
			log.Error("httpd: failed to create main page template: %s", err.Error())
			os.Exit(1)
		}

		if err := t.Execute(w, map[string]interface{}{
			"Version":    Version,
			"Commit":     Commit,
			"Configured": len(site.LoadPoints()),
		}); err != nil {
			log.Error("httpd: failed to render main page: %s", err.Error())
			os.Exit(1)
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
		log.Error("httpd: failed to encode JSON: %v", err)
	}
}

func jsonResult(w http.ResponseWriter, res interface{}) {
	w.WriteHeader(http.StatusOK)
	jsonWrite(w, map[string]interface{}{"result": res})
}

func jsonError(w http.ResponseWriter, status int, err error) {
	w.WriteHeader(status)
	jsonWrite(w, map[string]interface{}{"error": err.Error()})
}

// healthHandler returns current charge mode
func healthHandler(site site.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !site.Healthy() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "OK")
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

// targetSoCHandler updates target soc
func targetSoCHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		soc, err := strconv.ParseInt(vars["value"], 10, 32)
		if err == nil {
			lp.SetTargetSoC(int(soc))
		} else {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonResult(w, lp.GetTargetSoC())
	}
}

// minSoCHandler updates minimum soc
func minSoCHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		soc, err := strconv.ParseInt(vars["value"], 10, 32)
		if err == nil {
			lp.SetMinSoC(int(soc))
		} else {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonResult(w, lp.GetMinSoC())
	}
}

// minCurrentHandler updates minimum current
func minCurrentHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		current, err := strconv.ParseFloat(vars["value"], 64)
		if err == nil {
			lp.SetMinCurrent(current)
		} else {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonResult(w, lp.GetMinCurrent())
	}
}

// maxCurrentHandler updates maximum current
func maxCurrentHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		current, err := strconv.ParseFloat(vars["value"], 64)
		if err == nil {
			lp.SetMaxCurrent(current)
		} else {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		jsonResult(w, lp.GetMaxCurrent())
	}
}

// phasesHandler updates minimum soc
func phasesHandler(lp loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		phases, err := strconv.ParseInt(vars["value"], 10, 32)
		if err == nil {
			err = lp.SetPhases(int(phases))
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

func timezone() *time.Location {
	tz := os.Getenv("TZ")
	if tz == "" {
		tz = "Local"
	}

	loc, _ := time.LoadLocation(tz)
	return loc
}

// targetChargeHandler updates target soc
func targetChargeHandler(loadpoint loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		socS, ok := vars["soc"]
		socV, err := strconv.ParseInt(socS, 10, 32)

		if !ok || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		timeS, ok := vars["time"]
		timeV, err := time.ParseInLocation("2006-01-02T15:04:05", timeS, timezone())

		if !ok || err != nil {
			jsonError(w, http.StatusBadRequest, err)
			return
		}

		loadpoint.SetTargetCharge(timeV, int(socV))

		res := struct {
			SoC  int64     `json:"soc"`
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

// vehicleRemoveHandler removes vehicle
func vehicleRemoveHandler(loadpoint loadpoint.API) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loadpoint.SetVehicle(nil)
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
