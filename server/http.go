package server

import (
	"fmt"
	"net/http"
	"time"

	eapi "github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/telemetry"
	"github.com/go-http-utils/etag"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

type route struct {
	Method      string
	Pattern     string
	HandlerFunc http.HandlerFunc
}

func (r route) Methods() []string {
	if r.Method == http.MethodGet {
		return []string{r.Method}
	}
	return []string{r.Method, http.MethodOptions}
}

// HTTPd wraps an http.Server and adds the root router
type HTTPd struct {
	*http.Server
}

// NewHTTPd creates HTTP server with configured routes for loadpoint
func NewHTTPd(addr string, hub *SocketHub) *HTTPd {
	router := mux.NewRouter().StrictSlash(true)

	// websocket
	router.HandleFunc("/ws", socketHandler(hub))

	// static - individual handlers per root and folders
	static := router.PathPrefix("/").Subrouter()
	static.Use(handlers.CompressHandler)
	static.Use(handlers.CompressHandler, func(h http.Handler) http.Handler {
		return etag.Handler(h, false)
	})

	// allow requesting http assets from a non-private host. see https://developer.chrome.com/blog/cors-rfc1918-feedback?hl=de#step-2:-sending-preflight-requests-with-a-special-header
	static.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Private-Network", "true")
			next.ServeHTTP(w, r)
		})
	})

	static.HandleFunc("/", indexHandler())
	for _, dir := range []string{"assets", "meta"} {
		static.PathPrefix("/" + dir).Handler(http.FileServer(http.FS(assets.Web)))
	}
	static.PathPrefix("/i18n").Handler(http.StripPrefix("/i18n", http.FileServer(http.FS(assets.I18n))))

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

// RegisterSiteHandlers connects the http handlers to the site
func (s *HTTPd) RegisterSiteHandlers(site site.API, cache *util.Cache) {
	router := s.Server.Handler.(*mux.Router)

	// api
	api := router.PathPrefix("/api").Subrouter()
	api.Use(jsonHandler)
	api.Use(handlers.CompressHandler)
	api.Use(handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type"}),
	))

	// site api
	routes := map[string]route{
		"health":                  {"GET", "/health", healthHandler(site)},
		"state":                   {"GET", "/state", stateHandler(cache)},
		"config":                  {"GET", "/config/templates/{class:[a-z]+}", templatesHandler},
		"products":                {"GET", "/config/products/{class:[a-z]+}", productsHandler},
		"devices":                 {"GET", "/config/devices/{class:[a-z]+}", devicesHandler},
		"device":                  {"GET", "/config/devices/{class:[a-z]+}/{id:[0-9.]+}", deviceConfigHandler},
		"devicestatus":            {"GET", "/config/devices/{class:[a-z]+}/{name:[a-zA-Z0-9_.:-]+}/status", deviceStatusHandler},
		"site":                    {"GET", "/config/site", siteHandler(site)},
		"dirty":                   {"GET", "/config/dirty", boolGetHandler(ConfigDirty)},
		"updatesite":              {"PUT", "/config/site", updateSiteHandler(site)},
		"newdevice":               {"POST", "/config/devices/{class:[a-z]+}", newDeviceHandler},
		"updatedevice":            {"PUT", "/config/devices/{class:[a-z]+}/{id:[0-9.]+}", updateDeviceHandler},
		"deletedevice":            {"DELETE", "/config/devices/{class:[a-z]+}/{id:[0-9.]+}", deleteDeviceHandler},
		"testconfig":              {"POST", "/config/test/{class:[a-z]+}", testConfigHandler},
		"testmerged":              {"POST", "/config/test/{class:[a-z]+}/merge/{id:[0-9.]+}", testConfigHandler},
		"buffersoc":               {"POST", "/buffersoc/{value:[0-9.]+}", floatHandler(site.SetBufferSoc, site.GetBufferSoc)},
		"bufferstartsoc":          {"POST", "/bufferstartsoc/{value:[0-9.]+}", floatHandler(site.SetBufferStartSoc, site.GetBufferStartSoc)},
		"batterydischargecontrol": {"POST", "/batterydischargecontrol/{value:[a-z]+}", boolHandler(site.SetBatteryDischargeControl, site.GetBatteryDischargeControl)},
		"prioritysoc":             {"POST", "/prioritysoc/{value:[0-9.]+}", floatHandler(site.SetPrioritySoc, site.GetPrioritySoc)},
		"residualpower":           {"POST", "/residualpower/{value:[-0-9.]+}", floatHandler(site.SetResidualPower, site.GetResidualPower)},
		"smartcost":               {"POST", "/smartcostlimit/{value:[-0-9.]+}", updateSmartCostLimit(site)},
		"tariff":                  {"GET", "/tariff/{tariff:[a-z]+}", tariffHandler(site)},
		"sessions":                {"GET", "/sessions", sessionHandler},
		"updatesession":           {"PUT", "/session/{id:[0-9]+}", updateSessionHandler},
		"deletesession":           {"DELETE", "/session/{id:[0-9]+}", deleteSessionHandler},
		"telemetry":               {"GET", "/settings/telemetry", boolGetHandler(telemetry.Enabled)},
		"telemetry2":              {"POST", "/settings/telemetry/{value:[a-z]+}", boolHandler(telemetry.Enable, telemetry.Enabled)},
	}

	for _, r := range routes {
		api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
	}

	// vehicle api
	vehicles := map[string]route{
		"minsoc":   {"POST", "/vehicles/{name:[a-zA-Z0-9_.:-]+}/minsoc/{value:[0-9]+}", minSocHandler(site)},
		"limitsoc": {"POST", "/vehicles/{name:[a-zA-Z0-9_.:-]+}/limitsoc/{value:[0-9]+}", limitSocHandler(site)},
		"plan":     {"POST", "/vehicles/{name:[a-zA-Z0-9_.:-]+}/plan/soc/{value:[0-9]+}/{time:[0-9TZ:.-]+}", planSocHandler(site)},
		"plan2":    {"DELETE", "/vehicles/{name:[a-zA-Z0-9_.:-]+}/plan/soc", planSocRemoveHandler(site)},

		// config ui
		// "mode":       {"POST", "/mode/{value:[a-z]+}", chargeModeHandler(v)},
		// "mincurrent": {"POST", "/mincurrent/{value:[0-9.]+}", floatHandler(pass(v.SetMinCurrent), v.GetMinCurrent)},
		// "maxcurrent": {"POST", "/maxcurrent/{value:[0-9.]+}", floatHandler(pass(v.SetMaxCurrent), v.GetMaxCurrent)},
		// "phases":     {"POST", "/phases/{value:[0-9]+}", intHandler(pass(v.SetMinSoc), v.GetMinSoc)},
	}

	for _, r := range vehicles {
		api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
	}

	// loadpoint api
	for id, lp := range site.Loadpoints() {
		api := api.PathPrefix(fmt.Sprintf("/loadpoints/%d", id+1)).Subrouter()

		routes := map[string]route{
			"mode":             {"POST", "/mode/{value:[a-z]+}", handler(eapi.ChargeModeString, pass(lp.SetMode), lp.GetMode)},
			"limitsoc":         {"POST", "/limitsoc/{value:[0-9]+}", intHandler(pass(lp.SetLimitSoc), lp.GetLimitSoc)},
			"limitenergy":      {"POST", "/limitenergy/{value:[0-9.]+}", floatHandler(pass(lp.SetLimitEnergy), lp.GetLimitEnergy)},
			"mincurrent":       {"POST", "/mincurrent/{value:[0-9.]+}", floatHandler(lp.SetMinCurrent, lp.GetMinCurrent)},
			"maxcurrent":       {"POST", "/maxcurrent/{value:[0-9.]+}", floatHandler(lp.SetMaxCurrent, lp.GetMaxCurrent)},
			"phases":           {"POST", "/phases/{value:[0-9]+}", intHandler(lp.SetPhases, lp.GetPhases)},
			"plan":             {"GET", "/plan", planHandler(lp)},
			"planpreview":      {"GET", "/plan/preview/{type:(?:soc|energy)}/{value:[0-9.]+}/{time:[0-9TZ:.-]+}", planPreviewHandler(lp)},
			"planenergy":       {"POST", "/plan/energy/{value:[0-9.]+}/{time:[0-9TZ:.-]+}", planEnergyHandler(lp)},
			"planenergy2":      {"DELETE", "/plan/energy", planRemoveHandler(lp)},
			"vehicle":          {"POST", "/vehicle/{name:[a-zA-Z0-9_.:-]+}", vehicleSelectHandler(site, lp)},
			"vehicle2":         {"DELETE", "/vehicle", vehicleRemoveHandler(lp)},
			"vehicleDetect":    {"PATCH", "/vehicle", vehicleDetectHandler(lp)},
			"remotedemand":     {"POST", "/remotedemand/{demand:[a-z]+}/{source:[0-9a-zA-Z_-]+}", remoteDemandHandler(lp)},
			"enableThreshold":  {"POST", "/enable/threshold/{value:-?[0-9.]+}", floatHandler(pass(lp.SetEnableThreshold), lp.GetEnableThreshold)},
			"disableThreshold": {"POST", "/disable/threshold/{value:-?[0-9.]+}", floatHandler(pass(lp.SetDisableThreshold), lp.GetDisableThreshold)},
			"smartCostLimit":   {"POST", "/smartcostlimit/{value:[0-9.]+}", floatHandler(pass(lp.SetSmartCostLimit), lp.GetSmartCostLimit)},
			// "priority":         {"POST", "/priority/{value:[0-9.]+}", floatHandler(pass(lp.SetPriority), lp.GetPriority)},
		}

		for _, r := range routes {
			api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
		}
	}
}

// RegisterShutdownHandler connects the http handlers to the site
func (s *HTTPd) RegisterShutdownHandler(callback func()) {
	router := s.Server.Handler.(*mux.Router)

	// api
	api := router.PathPrefix("/api").Subrouter()
	api.Use(jsonHandler)
	api.Use(handlers.CompressHandler)
	api.Use(handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type"}),
	))

	// site api
	routes := map[string]route{
		"shutdown": {"POST", "/shutdown", func(w http.ResponseWriter, r *http.Request) {
			callback()
			w.WriteHeader(http.StatusNoContent)
		}},
	}

	for _, r := range routes {
		api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
	}
}
