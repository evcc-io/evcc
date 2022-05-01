package server

import (
	"fmt"
	"io/fs"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/util"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Assets is the embedded assets file system
var Assets fs.FS

type route struct {
	Methods     []string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

//lint:ignore U1000 if needed
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

// HTTPd wraps an http.Server and adds the root router
type HTTPd struct {
	*http.Server
}

// NewHTTPd creates HTTP server with configured routes for loadpoint
func NewHTTPd(addr string, site site.API, hub *SocketHub, cache *util.Cache) *HTTPd {
	routes := map[string]route{
		"health": {[]string{"GET"}, "/health", healthHandler(site)},
		"state":  {[]string{"GET"}, "/state", stateHandler(cache)},
	}

	router := mux.NewRouter().StrictSlash(true)

	// websocket
	router.HandleFunc("/ws", socketHandler(hub))

	// static - individual handlers per root and folders
	static := router.PathPrefix("/").Subrouter()
	static.Use(handlers.CompressHandler)

	static.HandleFunc("/", indexHandler(site))
	for _, dir := range []string{"assets", "meta"} {
		static.PathPrefix("/" + dir).Handler(http.FileServer(http.FS(Assets)))
	}

	// api
	api := router.PathPrefix("/api").Subrouter()
	api.Use(jsonHandler)
	api.Use(handlers.CompressHandler)
	api.Use(handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type"}),
	))

	// site api
	for _, r := range routes {
		api.Methods(r.Methods...).Path(r.Pattern).Handler(r.HandlerFunc)
	}

	// loadpoint api
	for id, lp := range site.LoadPoints() {
		lpAPI := api.PathPrefix(fmt.Sprintf("/loadpoints/%d", id)).Subrouter()

		routes := map[string]route{
			"mode":          {[]string{"POST", "OPTIONS"}, "/mode/{value:[a-z]+}", chargeModeHandler(lp)},
			"targetsoc":     {[]string{"POST", "OPTIONS"}, "/targetsoc/{value:[0-9]+}", targetSoCHandler(lp)},
			"minsoc":        {[]string{"POST", "OPTIONS"}, "/minsoc/{value:[0-9]+}", minSoCHandler(lp)},
			"mincurrent":    {[]string{"POST", "OPTIONS"}, "/mincurrent/{value:[0-9]+}", minCurrentHandler(lp)},
			"maxcurrent":    {[]string{"POST", "OPTIONS"}, "/maxcurrent/{value:[0-9]+}", maxCurrentHandler(lp)},
			"phases":        {[]string{"POST", "OPTIONS"}, "/phases/{value:[0-9]+}", phasesHandler(lp)},
			"targetcharge":  {[]string{"POST", "OPTIONS"}, "/targetcharge/{soc:[0-9]+}/{time:[0-9TZ:-]+}", targetChargeHandler(lp)},
			"targetcharge2": {[]string{"DELETE", "OPTIONS"}, "/targetcharge", targetChargeRemoveHandler(lp)},
			"vehicle":       {[]string{"DELETE", "OPTIONS"}, "/vehicle", vehicleRemoveHandler(lp)},
			"remotedemand":  {[]string{"POST", "OPTIONS"}, "/remotedemand/{demand:[a-z]+}/{source::[0-9a-zA-Z_-]+}", remoteDemandHandler(lp)},
		}

		for _, r := range routes {
			lpAPI.Methods(r.Methods...).Path(r.Pattern).Handler(r.HandlerFunc)
		}
	}

	srv := &HTTPd{
		Server: &http.Server{
			Addr:         addr,
			Handler:      router,
			ReadTimeout:  5 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  120 * time.Second,
			ErrorLog:     log.ERROR,
		},
	}
	srv.SetKeepAlivesEnabled(true)

	return srv
}

// Router returns the main router
func (s *HTTPd) Router() *mux.Router {
	return s.Handler.(*mux.Router)
}
