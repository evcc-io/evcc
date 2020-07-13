package server

import (
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core"
	"github.com/andig/evcc/util"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

//go:generate esc -o assets.go -pkg server -modtime 1566640112 -ignore .DS_Store -prefix ../assets ../assets

const (
	liveAssets = false
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

type route struct {
	Methods     []string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// site is the minimal interface for accessing site methods
type site interface {
	GetMode() api.ChargeMode
	SetMode(api.ChargeMode)
	Configuration() core.SiteConfiguration
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

func indexHandler(links []MenuConfig, liveAssets bool) http.HandlerFunc {
	indexTemplate, err := FSString(liveAssets, "/index.html")
	if err != nil {
		log.FATAL.Fatal("httpd: failed to load embedded template: " + err.Error())
	}

	t, err := template.New("evcc").Delims("<<", ">>").Parse(indexTemplate)
	if err != nil {
		log.FATAL.Fatal("httpd: failed to create main page template: ", err.Error())
	}

	_, debug := _escData["/js/debug.js"]

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// w.Header().Set("Content-Type", "text/html; charset=UTF-8")
		// w.WriteHeader(http.StatusOK)

		if err := t.Execute(w, map[string]interface{}{
			"Debug": debug,
			"Links": links,
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

// StateHandler returns current charge mode
func StateHandler(cache *util.Cache) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := cache.State()
		jsonResponse(w, r, res)
	}
}

// CurrentChargeModeHandler returns current charge mode
func CurrentChargeModeHandler(site site) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		res := chargeModeJSON{Mode: site.GetMode()}
		jsonResponse(w, r, res)
	}
}

// ChargeModeHandler updates charge mode
func ChargeModeHandler(site site) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		mode, ok := vars["mode"]
		if !ok {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		site.SetMode(api.ChargeMode(mode))

		res := chargeModeJSON{Mode: site.GetMode()}
		jsonResponse(w, r, res)
	}
}

// SocketHandler attaches websocket handler to uri
func SocketHandler(hub *SocketHub) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ServeWebsocket(hub, w, r)
	}
}

// NewHTTPd creates HTTP server with configured routes for loadpoint
func NewHTTPd(url string, links []MenuConfig, site site, hub *SocketHub, cache *util.Cache) *http.Server {
	var routes = []route{{
		[]string{"GET"}, "/health", HealthHandler(),
	}, {
		[]string{"GET"}, "/config", ConfigHandler(site),
	}, {
		[]string{"GET"}, "/state", StateHandler(cache),
	}, {
		[]string{"GET"}, "/mode", CurrentChargeModeHandler(site),
	}, {
		[]string{"PUT", "POST", "OPTIONS"}, "/mode/{mode:[a-z]+}", ChargeModeHandler(site),
	}}

	router := mux.NewRouter().StrictSlash(true)

	// websocket
	router.HandleFunc("/ws", SocketHandler(hub))

	// static - individual handlers per root and folders
	static := router.PathPrefix("/").Subrouter()
	static.Use(handlers.CompressHandler)

	static.HandleFunc("/", indexHandler(links, liveAssets))
	for _, folder := range []string{"js", "css", "webfonts", "ico"} {
		prefix := fmt.Sprintf("/%s/", folder)
		static.PathPrefix(prefix).Handler(http.StripPrefix(prefix, http.FileServer(Dir(liveAssets, prefix))))
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

	for _, r := range routes {
		api.
			Methods(r.Methods...).
			Path(r.Pattern).
			Handler(r.HandlerFunc) // routeLogger
	}

	srv := &http.Server{
		Addr:         url,
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
		ErrorLog:     log.ERROR,
	}
	srv.SetKeepAlivesEnabled(true)

	return srv
}
