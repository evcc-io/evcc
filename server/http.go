package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/test"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// MenuConfig is used to inject the menu configuration into the UI
type MenuConfig struct {
	Title    string
	Subtitle string
	Img      string
	Iframe   string
	Link     string
}

type chargeModeJSON struct {
	Mode api.ChargeMode `json:"mode"`
}

type targetSoCJSON struct {
	TargetSoC int `json:"targetSoC"`
}

type route struct {
	Methods     []string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// site is the minimal interface for accessing site methods
type site interface {
	Configuration() core.SiteConfiguration
	LoadPoints() []*core.LoadPoint
	loadpoint
}

// loadpoint is the minimal interface for accessing loadpoint methods
type loadpoint interface {
	GetMode() api.ChargeMode
	SetMode(api.ChargeMode)
	GetTargetSoC() int
	SetTargetSoC(targetSoC int)
}

// routeLogger traces matched routes including their executing time
func routeLogger(inner http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		inner.ServeHTTP(w, r)
		log.TRACE.Printf(
			"%s\t%s\t%s",
			r.Method,
			r.RequestURI,
			time.Since(start),
		)
	}
}

func indexHandler(links []MenuConfig, site site, useLocal bool) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html; charset=UTF-8")

		indexTemplate, err := FSString(useLocal, "/index.html")
		if err != nil {
			log.FATAL.Fatal("httpd: failed to load embedded template: " + err.Error())
		}

		t, err := template.New("evcc").Delims("<<", ">>").Parse(indexTemplate)
		if err != nil {
			log.FATAL.Fatal("httpd: failed to create main page template: ", err.Error())
		}

		if err := t.Execute(w, map[string]interface{}{
			"Version":    Version,
			"Commit":     Commit,
			"Links":      links,
			"Configured": len(site.LoadPoints()),
			"Tag":        time.Now().Unix(),
		}); err != nil {
			log.ERROR.Println("httpd: failed to render main page: ", err.Error())
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

func jsonResponse(w http.ResponseWriter, r *http.Request, content interface{}) {
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(content); err != nil {
		log.ERROR.Printf("httpd: failed to encode JSON: %v", err)
	}
}

// HealthHandler returns current charge mode
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := struct{ OK bool }{OK: true}
		jsonResponse(w, r, res)
	}
}

// ConfigHandler returns current charge mode
func ConfigHandler(site site) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := site.Configuration()
		jsonResponse(w, r, res)
	}
}

// TemplatesHandler returns current charge mode
func TemplatesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		class, ok := vars["class"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		type template = struct {
			Name   string `json:"name"`
			Sample string `json:"template"`
		}

		res := make([]template, 0)
		for _, conf := range test.ConfigTemplates(class) {
			typedSample := fmt.Sprintf("type: %s\n%s", conf.Type, conf.Sample)
			t := template{
				Name:   conf.Name,
				Sample: typedSample,
			}
			res = append(res, t)
		}

		jsonResponse(w, r, res)
	}
}

// StateHandler returns current charge mode
func StateHandler(cache *util.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := cache.State()
		for _, k := range []string{"availableVersion", "releaseNotes"} {
			delete(res, k)
		}
		jsonResponse(w, r, res)
	}
}

// CurrentChargeModeHandler returns current charge mode
func CurrentChargeModeHandler(loadpoint loadpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := chargeModeJSON{Mode: loadpoint.GetMode()}
		jsonResponse(w, r, res)
	}
}

// ChargeModeHandler updates charge mode
func ChargeModeHandler(loadpoint loadpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		modeS, ok := vars["mode"]
		mode := api.ChargeModeString(modeS)
		if mode == "" || string(mode) != modeS || !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		loadpoint.SetMode(mode)

		res := chargeModeJSON{Mode: loadpoint.GetMode()}
		jsonResponse(w, r, res)
	}
}

// CurrentTargetSoCHandler returns current target soc
func CurrentTargetSoCHandler(loadpoint loadpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := targetSoCJSON{TargetSoC: loadpoint.GetTargetSoC()}
		jsonResponse(w, r, res)
	}
}

// TargetSoCHandler updates target soc
func TargetSoCHandler(loadpoint loadpoint) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		socS, ok := vars["soc"]
		soc, err := strconv.ParseInt(socS, 10, 32)

		if !ok || err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		loadpoint.SetTargetSoC(int(soc))

		res := targetSoCJSON{TargetSoC: loadpoint.GetTargetSoC()}
		jsonResponse(w, r, res)
	}
}

// SocketHandler attaches websocket handler to uri
func SocketHandler(hub *SocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ServeWebsocket(hub, w, r)
	}
}

// applyRouteHandler applies route with given handler
func applyRouteHandler(router *mux.Router, r route, handler http.HandlerFunc) {
	router.Methods(r.Methods...).Path(r.Pattern).Handler(handler)
}

// HTTPd wraps an http.Server and adds the root router
type HTTPd struct {
	*http.Server
	*mux.Router
}

// NewHTTPd creates HTTP server with configured routes for loadpoint
func NewHTTPd(url string, links []MenuConfig, site site, hub *SocketHub, cache *util.Cache) *HTTPd {
	var routes = map[string]route{
		"health":       {[]string{"GET"}, "/health", HealthHandler()},
		"config":       {[]string{"GET"}, "/config", ConfigHandler(site)},
		"templates":    {[]string{"GET"}, "/config/templates/{class:[a-z]+}", TemplatesHandler()},
		"state":        {[]string{"GET"}, "/state", StateHandler(cache)},
		"getmode":      {[]string{"GET"}, "/mode", CurrentChargeModeHandler(site)},
		"setmode":      {[]string{"POST", "OPTIONS"}, "/mode/{mode:[a-z]+}", ChargeModeHandler(site)},
		"gettargetsoc": {[]string{"GET"}, "/targetsoc", CurrentTargetSoCHandler(site)},
		"settargetsoc": {[]string{"POST", "OPTIONS"}, "/targetsoc/{soc:[0-9]+}", TargetSoCHandler(site)},
	}

	router := mux.NewRouter().StrictSlash(true)

	// websocket
	router.HandleFunc("/ws", SocketHandler(hub))

	// static - individual handlers per root and folders
	static := router.PathPrefix("/").Subrouter()
	static.Use(handlers.CompressHandler)

	static.HandleFunc("/", indexHandler(links, site, useLocalAssets))
	for _, folder := range []string{"js", "css", "webfonts", "ico"} {
		prefix := fmt.Sprintf("/%s/", folder)
		static.PathPrefix(prefix).Handler(http.StripPrefix(prefix, http.FileServer(Dir(useLocalAssets, prefix))))
	}

	// api
	api := router.PathPrefix("/api").Subrouter()
	api.Use(jsonHandler)
	api.Use(handlers.CompressHandler)
	api.Use(handlers.CORS(
		handlers.AllowedHeaders([]string{
			"Accept", "Accept-Language", "Content-Language", "Content-Type", "Origin",
		}),
	))

	// site api
	for _, r := range routes {
		api.Methods(r.Methods...).Path(r.Pattern).Handler(r.HandlerFunc)
	}

	// loadpoint api
	for id, lp := range site.LoadPoints() {
		subAPI := api.PathPrefix(fmt.Sprintf("/loadpoints/%d", id)).Subrouter()
		applyRouteHandler(subAPI, routes["getmode"], CurrentChargeModeHandler(lp))
		applyRouteHandler(subAPI, routes["setmode"], ChargeModeHandler(lp))
		applyRouteHandler(subAPI, routes["gettargetsoc"], CurrentTargetSoCHandler(lp))
		applyRouteHandler(subAPI, routes["settargetsoc"], TargetSoCHandler(lp))
	}

	srv := &HTTPd{
		Server: &http.Server{
			Addr:         url,
			Handler:      router,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
			ErrorLog:     log.ERROR,
		},
		Router: router,
	}
	srv.SetKeepAlivesEnabled(true)

	return srv
}
