package server

import (
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"time"

	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/encode"
	"github.com/gorilla/mux"
)

var displayLoadpointFields = []string{
	"name", "title", "disabled", "mode", "alwaysCharge", "chargePower", "charging", "connected",
	"sessionEnergy", "chargedEnergy", "chargeDuration", "phasesActive", "sessionSolarPercentage",
	"vehicleSoc", "vehicleTitle", "pvAction", "pvRemaining", "planActive", "smartCostActive",
	"effectiveMinCurrent", "minCurrent", "phasesConfigured", "chargerPhases1p3p",
}

//go:embed display/static/*
var displayFiles embed.FS

// NewDisplayHTTPd creates the read-only garage display server.
func NewDisplayHTTPd(addr string, cache *util.ParamCache) *HTTPd {
	display, err := fs.Sub(displayFiles, "display/static")
	if err != nil {
		panic(fmt.Errorf("display assets: %w", err))
	}

	router := mux.NewRouter()
	router.Handle("/api/state", http.HandlerFunc(displayStateHandler(cache))).Methods(http.MethodGet)
	router.HandleFunc("/i18n/{locale:[a-zA-Z0-9-]+\\.json}", displayTranslationHandler).Methods(http.MethodGet)
	router.HandleFunc("/fonts/{name:Montserrat-(?:Medium|Bold)\\.woff2}", displayFontHandler).Methods(http.MethodGet)

	files := http.FileServer(http.FS(display))
	router.PathPrefix("/").Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Cache-Control", "no-cache")
		files.ServeHTTP(w, r)
	}))

	log := util.NewLogger("display")
	srv := &HTTPd{
		Server: &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 30 * time.Second,
			IdleTimeout:  120 * time.Second,
			ErrorLog:     log.ERROR,
		},
	}
	srv.SetKeepAlivesEnabled(true)

	return srv
}

func displayTranslationHandler(w http.ResponseWriter, r *http.Request) {
	if assets.I18n == nil {
		http.Error(w, "translations unavailable", http.StatusInternalServerError)
		return
	}

	locale := mux.Vars(r)["locale"]
	data, err := fs.ReadFile(assets.I18n, locale)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			http.NotFound(w, r)
			return
		}
		http.Error(w, "translations unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache")
	_, _ = w.Write(data)
}

func displayStateHandler(cache *util.ParamCache) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		state := cache.State(encode.NewEncoder(encode.WithDuration()))
		loadpoints, ok := state["loadpoints"].([]map[string]any)
		if !ok {
			http.Error(w, "display state unavailable", http.StatusInternalServerError)
			return
		}

		displayLoadpoints := make([]map[string]any, len(loadpoints))
		for i, loadpoint := range loadpoints {
			displayLoadpoints[i] = selectStateFields(loadpoint, displayLoadpointFields)
		}

		response := selectStateFields(state, []string{
			"greenShareLoadpoints", "pvPower", "homePower", "batterySoc", "batteryPower", "gridPower",
		})
		response["loadpoints"] = displayLoadpoints
		battery, err := nestedStateFields(state, "battery", []string{"soc", "power"})
		if err != nil {
			http.Error(w, "display state unavailable", http.StatusInternalServerError)
			return
		}
		grid, err := nestedStateFields(state, "grid", []string{"power"})
		if err != nil {
			http.Error(w, "display state unavailable", http.StatusInternalServerError)
			return
		}
		response["battery"] = battery
		response["grid"] = grid

		jsonWrite(w, response)
	}
}

func nestedStateFields(state map[string]any, key string, fields []string) (map[string]any, error) {
	value, ok := state[key]
	if !ok || value == nil {
		return selectStateFields(nil, fields), nil
	}

	nested, ok := value.(map[string]any)
	if ok {
		return selectStateFields(nested, fields), nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("encode %s state: %w", key, err)
	}
	if err := json.Unmarshal(data, &nested); err != nil {
		return nil, fmt.Errorf("decode %s state: %w", key, err)
	}
	return selectStateFields(nested, fields), nil
}

func selectStateFields(state map[string]any, fields []string) map[string]any {
	selected := make(map[string]any, len(fields))
	for _, field := range fields {
		selected[field] = state[field]
	}
	return selected
}

func displayFontHandler(w http.ResponseWriter, r *http.Request) {
	name := mux.Vars(r)["name"]
	if assets.Web == nil {
		http.NotFound(w, r)
		return
	}

	fontName := strings.TrimSuffix(name, ".woff2")
	matches, err := fs.Glob(assets.Web, "assets/"+fontName+"-*.woff2")
	if err != nil {
		http.Error(w, "font unavailable", http.StatusInternalServerError)
		return
	}
	if len(matches) != 1 {
		http.NotFound(w, r)
		return
	}

	font, err := assets.Web.Open(matches[0])
	if err != nil {
		http.Error(w, "font unavailable", http.StatusInternalServerError)
		return
	}
	defer font.Close()

	stat, err := fs.Stat(assets.Web, matches[0])
	if err != nil {
		http.Error(w, "font unavailable", http.StatusInternalServerError)
		return
	}
	seeker, ok := font.(io.ReadSeeker)
	if !ok {
		http.Error(w, "font unavailable", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "font/woff2")
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("Cache-Control", "no-cache")
	http.ServeContent(w, r, name, stat.ModTime(), seeker)
}
