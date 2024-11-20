package server

import (
	"fmt"
	"net/http"
	"time"

	eapi "github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/server/assets"
	"github.com/evcc-io/evcc/server/eebus"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/auth"
	"github.com/evcc-io/evcc/util/config"
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
func (s *HTTPd) RegisterSiteHandlers(site site.API, valueChan chan<- util.Param) {
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
		"buffersoc":               {"POST", "/buffersoc/{value:[0-9.]+}", floatHandler(site.SetBufferSoc, site.GetBufferSoc)},
		"bufferstartsoc":          {"POST", "/bufferstartsoc/{value:[0-9.]+}", floatHandler(site.SetBufferStartSoc, site.GetBufferStartSoc)},
		"batterydischargecontrol": {"POST", "/batterydischargecontrol/{value:[01truefalse]+}", boolHandler(site.SetBatteryDischargeControl, site.GetBatteryDischargeControl)},
		"batterygridcharge":       {"POST", "/batterygridchargelimit/{value:-?[0-9.]+}", floatPtrHandler(pass(site.SetBatteryGridChargeLimit), site.GetBatteryGridChargeLimit)},
		"batterygridchargedelete": {"DELETE", "/batterygridchargelimit", floatPtrHandler(pass(site.SetBatteryGridChargeLimit), site.GetBatteryGridChargeLimit)},
		"prioritysoc":             {"POST", "/prioritysoc/{value:[0-9.]+}", floatHandler(site.SetPrioritySoc, site.GetPrioritySoc)},
		"residualpower":           {"POST", "/residualpower/{value:-?[0-9.]+}", floatHandler(site.SetResidualPower, site.GetResidualPower)},
		"smartcost":               {"POST", "/smartcostlimit/{value:-?[0-9.]+}", updateSmartCostLimit(site)},
		"smartcostdelete":         {"DELETE", "/smartcostlimit", updateSmartCostLimit(site)},
		"tariff":                  {"GET", "/tariff/{tariff:[a-z]+}", tariffHandler(site)},
		"sessions":                {"GET", "/sessions", sessionHandler},
		"updatesession":           {"PUT", "/session/{id:[0-9]+}", updateSessionHandler},
		"deletesession":           {"DELETE", "/session/{id:[0-9]+}", deleteSessionHandler},
		"telemetry":               {"GET", "/settings/telemetry", getHandler(telemetry.Enabled)},
		"telemetry2":              {"POST", "/settings/telemetry/{value:[01truefalse]+}", boolHandler(telemetry.Enable, telemetry.Enabled)},
	}

	for _, r := range routes {
		api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
	}

	// config ui (secured)
	configApi := api.PathPrefix("/config").Subrouter()

	// TODO clarify location of site config
	configRoutes := map[string]route{
		"site":       {"GET", "/site", siteHandler(site)},
		"updatesite": {"PUT", "/site", updateSiteHandler(site)},
	}

	for _, r := range configRoutes {
		configApi.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
	}

	// vehicle api
	vehicles := map[string]route{
		"minsoc":   {"POST", "/vehicles/{name:[a-zA-Z0-9_.:-]+}/minsoc/{value:[0-9]+}", minSocHandler(site)},
		"limitsoc": {"POST", "/vehicles/{name:[a-zA-Z0-9_.:-]+}/limitsoc/{value:[0-9]+}", limitSocHandler(site)},
		"plan":     {"POST", "/vehicles/{name:[a-zA-Z0-9_.:-]+}/plan/soc/{value:[0-9]+}/{time:[0-9TZ:.+-]+}", planSocHandler(site)},
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
			"planpreview":      {"GET", "/plan/preview/{type:(?:soc|energy)}/{value:[0-9.]+}/{time:[0-9TZ:.+-]+}", planPreviewHandler(lp)},
			"planenergy":       {"POST", "/plan/energy/{value:[0-9.]+}/{time:[0-9TZ:.+-]+}", planEnergyHandler(lp)},
			"planenergy2":      {"DELETE", "/plan/energy", planRemoveHandler(lp)},
			"vehicle":          {"POST", "/vehicle/{name:[a-zA-Z0-9_.:-]+}", vehicleSelectHandler(site, lp)},
			"vehicle2":         {"DELETE", "/vehicle", vehicleRemoveHandler(lp)},
			"vehicleDetect":    {"PATCH", "/vehicle", vehicleDetectHandler(lp)},
			"remotedemand":     {"POST", "/remotedemand/{demand:[a-z]+}/{source:[0-9a-zA-Z_-]+}", remoteDemandHandler(lp)},
			"enableThreshold":  {"POST", "/enable/threshold/{value:-?[0-9.]+}", floatHandler(pass(lp.SetEnableThreshold), lp.GetEnableThreshold)},
			"enableDelay":      {"POST", "/enable/delay/{value:[0-9]+}", durationHandler(pass(lp.SetEnableDelay), lp.GetEnableDelay)},
			"disableThreshold": {"POST", "/disable/threshold/{value:-?[0-9.]+}", floatHandler(pass(lp.SetDisableThreshold), lp.GetDisableThreshold)},
			"disableDelay":     {"POST", "/disable/delay/{value:[0-9]+}", durationHandler(pass(lp.SetDisableDelay), lp.GetDisableDelay)},
			"smartCost":        {"POST", "/smartcostlimit/{value:-?[0-9.]+}", floatPtrHandler(pass(lp.SetSmartCostLimit), lp.GetSmartCostLimit)},
			"smartCostDelete":  {"DELETE", "/smartcostlimit", floatPtrHandler(pass(lp.SetSmartCostLimit), lp.GetSmartCostLimit)},
			"priority":         {"POST", "/priority/{value:[0-9]+}", intHandler(pass(lp.SetPriority), lp.GetPriority)},
			"batteryBoost":     {"POST", "/batteryboost/{value:[01truefalse]}", boolHandler(lp.SetBatteryBoost, lp.GetBatteryBoost)},
		}

		for _, r := range routes {
			api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
		}
	}
}

// RegisterSystemHandler provides system level handlers
func (s *HTTPd) RegisterSystemHandler(valueChan chan<- util.Param, cache *util.Cache, auth auth.Auth, shutdown func()) {
	router := s.Server.Handler.(*mux.Router)

	// api
	api := router.PathPrefix("/api").Subrouter()
	api.Use(jsonHandler)
	api.Use(handlers.CompressHandler)
	api.Use(handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type"}),
	))

	{ // /api
		routes := map[string]route{
			"state": {"GET", "/state", stateHandler(cache)},
		}

		for _, r := range routes {
			api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
		}
	}

	{
		// api/auth
		api := api.PathPrefix("/auth").Subrouter()

		routes := map[string]route{
			"password": {"PUT", "/password", updatePasswordHandler(auth)},
			"auth":     {"GET", "/status", authStatusHandler(auth)},
			"login":    {"POST", "/login", loginHandler(auth)},
			"logout":   {"POST", "/logout", logoutHandler},
		}

		for _, r := range routes {
			api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
		}
	}

	{ // api/config
		api := api.PathPrefix("/config").Subrouter()
		api.Use(ensureAuthHandler(auth))

		routes := map[string]route{
			"templates":          {"GET", "/templates/{class:[a-z]+}", templatesHandler},
			"products":           {"GET", "/products/{class:[a-z]+}", productsHandler},
			"devices":            {"GET", "/devices/{class:[a-z]+}", devicesHandler},
			"device":             {"GET", "/devices/{class:[a-z]+}/{id:[0-9.]+}", deviceConfigHandler},
			"devicestatus":       {"GET", "/devices/{class:[a-z]+}/{name:[a-zA-Z0-9_.:-]+}/status", deviceStatusHandler},
			"dirty":              {"GET", "/dirty", getHandler(ConfigDirty)},
			"newdevice":          {"POST", "/devices/{class:[a-z]+}", newDeviceHandler},
			"updatedevice":       {"PUT", "/devices/{class:[a-z]+}/{id:[0-9.]+}", updateDeviceHandler},
			"deletedevice":       {"DELETE", "/devices/{class:[a-z]+}/{id:[0-9.]+}", deleteDeviceHandler},
			"testconfig":         {"POST", "/test/{class:[a-z]+}", testConfigHandler},
			"testmerged":         {"POST", "/test/{class:[a-z]+}/merge/{id:[0-9.]+}", testConfigHandler},
			"interval":           {"POST", "/interval/{value:[0-9.]+}", settingsSetDurationHandler(keys.Interval)},
			"updatesponsortoken": {"POST", "/sponsortoken", updateSponsortokenHandler},
			"deletesponsortoken": {"DELETE", "/sponsortoken", settingsDeleteHandler(keys.SponsorToken)},
		}

		// yaml handlers
		for key, fun := range map[string]func() (any, any){
			keys.EEBus:       func() (any, any) { return map[string]any{}, eebus.Config{} },
			keys.Hems:        func() (any, any) { return map[string]any{}, config.Typed{} },
			keys.Tariffs:     func() (any, any) { return map[string]any{}, globalconfig.Tariffs{} },
			keys.Messaging:   func() (any, any) { return map[string]any{}, globalconfig.Messaging{} },       // has default
			keys.ModbusProxy: func() (any, any) { return []map[string]any{}, []globalconfig.ModbusProxy{} }, // slice
			keys.Circuits:    func() (any, any) { return []map[string]any{}, []config.Named{} },             // slice
		} {
			other, struc := fun()
			routes[key] = route{Method: "GET", Pattern: "/" + key, HandlerFunc: settingsGetStringHandler(key)}
			routes["update"+key] = route{Method: "POST", Pattern: "/" + key, HandlerFunc: settingsSetYamlHandler(key, other, struc)}
			routes["delete"+key] = route{Method: "DELETE", Pattern: "/" + key, HandlerFunc: settingsDeleteHandler(key)}
		}

		// json handlers
		for key, fun := range map[string]func() any{
			keys.Network: func() any { return new(globalconfig.Network) }, // has default
			keys.Mqtt:    func() any { return new(globalconfig.Mqtt) },    // has default
			keys.Influx:  func() any { return new(globalconfig.Influx) },
		} {
			// routes[key] = route{Method: "GET", Pattern: "/" + key, HandlerFunc: settingsGetJsonHandler(key, fun())}
			routes["update"+key] = route{Method: "POST", Pattern: "/" + key, HandlerFunc: settingsSetJsonHandler(key, valueChan, fun())}
			routes["delete"+key] = route{Method: "DELETE", Pattern: "/" + key, HandlerFunc: settingsDeleteJsonHandler(key, valueChan, fun())}
		}

		for _, r := range routes {
			api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
		}
	}

	{ // api/system
		api := api.PathPrefix("/system").Subrouter()
		api.Use(ensureAuthHandler(auth))

		// system api
		routes := map[string]route{
			"log":      {"GET", "/log", logHandler},
			"logareas": {"GET", "/log/areas", logAreasHandler},
			"shutdown": {"POST", "/shutdown", func(w http.ResponseWriter, r *http.Request) {
				shutdown()
				w.WriteHeader(http.StatusNoContent)
			}},
		}

		for _, r := range routes {
			api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
		}
	}
}
