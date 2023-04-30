package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/telemetry"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/gorilla/handlers"
)

type route struct {
	Methods     []string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

// HTTPd wraps an http.Server and adds the root router
type HTTPd struct {
	*http.Server
}

// NewHTTPd creates HTTP server with configured routes for loadpoint
func NewHTTPd(addr string, hub *SocketHub) *HTTPd {
	router := chi.NewRouter()
	router.Use(middleware.Compress(5))
	router.Use(middleware.Recoverer)

	// websocket
	router.HandleFunc("/ws", socketHandler(hub))

	// static files
	router.HandleFunc("/", indexHandler())
	router.Handle("/assets", http.FileServer(http.FS(assets.Web)))
	router.Handle("/meta", http.FileServer(http.FS(assets.Web)))
	router.Handle("/i18n", http.StripPrefix("/i18n", http.FileServer(http.FS(assets.I18n))))

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
func (s *HTTPd) Router() *chi.Mux {
	return s.Handler.(*chi.Mux)
}

// RegisterSiteHandlers connects the http handlers to the site
func (s *HTTPd) RegisterSiteHandlers(site site.API, cache *util.Cache) {
	router := s.Server.Handler.(*chi.Mux)

	// api
	// api := router.PathPrefix("/api").Subrouter()
	// api.Use(jsonHandler)
	// api.Use(handlers.CompressHandler)
	// api.Use(handlers.CORS(
	// 	handlers.AllowedHeaders([]string{"Content-Type"}),
	// ))

	// site api
	router.Route("/api", func(r chi.Router) {
		r.Use(cors.Handler(cors.Options{
			// AllowedOrigins:   []string{"https://foo.com"}, // Use this to allow specific origin hosts
			AllowedOrigins: []string{"https://*", "http://*"},
			// AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           300, // Maximum value not ignored by any of major browsers
		}))

		r.Get("/health", healthHandler(site))
		r.Get("/state", stateHandler(cache))
		r.Get("/config/templates/{class:[a-z]+}", templatesHandler)
		r.Get("/config/products/{class:[a-z]+}", productsHandler)
		r.Post("/config/test/{class:[a-z]+}", testHandler)
		r.Post("/buffersoc/{value:[0-9.]+}", floatHandler(site.SetBufferSoc, site.GetBufferSoc))
		r.Post("/prioritysoc/{value:[0-9.]+}", floatHandler(site.SetPrioritySoc, site.GetPrioritySoc))
		r.Post("/residualpower/{value:[-0-9.]+}", floatHandler(site.SetResidualPower, site.GetResidualPower))
		r.Post("/smartcostlimit/{value:[-0-9.]+}", floatHandler(site.SetSmartCostLimit, site.GetSmartCostLimit))
		r.Get("/tariff/{tariff:[a-z]+}", tariffHandler(site))
		r.Get("/sessions", sessionHandler)
		r.Put("/session/{id:[0-9]+}", updateSessionHandler)
		r.Delete("/session/{id:[0-9]+}", deleteSessionHandler)
		r.Get("/settings/telemetry", boolGetHandler(telemetry.Enabled))
		r.Post("/settings/telemetry/{value:[a-z]+}", boolHandler(telemetry.Enable, telemetry.Enabled))

		// loadpoint api
		for id, lp := range site.Loadpoints() {
			r.Route(fmt.Sprintf("/loadpoints/%d", id+1), func(r chi.Router) {
				r.Post("/mode/{value:[a-z]+}", chargeModeHandler(lp))
				r.Post("/minsoc/{value:[0-9]+}", intHandler(pass(lp.SetMinSoc), lp.GetMinSoc))
				r.Post("/mincurrent/{value:[0-9.]+}", floatHandler(pass(lp.SetMinCurrent), lp.GetMinCurrent))
				r.Post("/maxcurrent/{value:[0-9.]+}", floatHandler(pass(lp.SetMaxCurrent), lp.GetMaxCurrent))
				r.Post("/phases/{value:[0-9]+}", phasesHandler(lp))
				r.Post("/target/energy/{value:[0-9.]+}", floatHandler(pass(lp.SetTargetEnergy), lp.GetTargetEnergy))
				r.Post("/target/soc/{value:[0-9]+}", intHandler(pass(lp.SetTargetSoc), lp.GetTargetSoc))
				r.Post("/target/time/{time:[0-9TZ:.-]+}", targetTimeHandler(lp))
				r.Delete("/target/time", targetTimeRemoveHandler(lp))
				r.Get("/target/plan", planHandler(lp))
				r.Post("/vehicle/{vehicle:[1-9][0-9]*}", vehicleHandler(site, lp))
				r.Delete("/vehicle", vehicleRemoveHandler(lp))
				r.Patch("/vehicle", vehicleDetectHandler(lp))
				r.Post("/remotedemand/{demand:[a-z]+}/{source::[0-9a-zA-Z_-]+}", remoteDemandHandler(lp))
				r.Post("/enable/threshold/{value:-?[0-9.]+}", floatHandler(pass(lp.SetEnableThreshold), lp.GetEnableThreshold))
				r.Post("/disable/threshold/{value:-?[0-9.]+}", floatHandler(pass(lp.SetDisableThreshold), lp.GetDisableThreshold))
			})
		}
	})
}

// RegisterShutdownHandler connects the http handlers to the site
func (s *HTTPd) RegisterShutdownHandler(callback func()) {
	router := s.Server.Handler.(*chi.Mux)

	// api
	api := router.PathPrefix("/api").Subrouter()
	api.Use(jsonHandler)
	api.Use(handlers.CompressHandler)
	api.Use(handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type"}),
	))

	// site api
	routes := map[string]route{
		"shutdown": {[]string{"POST", "OPTIONS"}, "/shutdown", func(w http.ResponseWriter, r *http.Request) {
			callback()
			w.WriteHeader(http.StatusNoContent)
		}},
	}

	for _, r := range routes {
		api.Methods(r.Methods...).Path(r.Pattern).Handler(r.HandlerFunc)
	}
}
