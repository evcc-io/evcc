package server

import (
	"fmt"
	"net/http"
	"os"
	"time"

	eapi "github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems/shm"
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
func NewHTTPd(addr string, hub *SocketHub, customCssFile string) *HTTPd {
	router := mux.NewRouter().StrictSlash(true)

	log := util.NewLogger("httpd")

	// log all requests
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.TRACE.Printf("%s %s", r.Method, r.URL.Path)
			next.ServeHTTP(w, r)
		})
	})

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

	if customCssFile != "" {
		log.WARN.Printf("❗ using custom CSS: %s", customCssFile)
		if _, err := os.Stat(customCssFile); os.IsNotExist(err) {
			log.FATAL.Fatalf("custom CSS file does not exist: %s", customCssFile)
		}
		static.HandleFunc("/custom.css", func(w http.ResponseWriter, r *http.Request) {
			// disable caching
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			w.Header().Set("Pragma", "no-cache")
			w.Header().Set("Expires", "0")
			http.ServeFile(w, r, customCssFile)
		})
	}

	static.HandleFunc("/", indexHandler(customCssFile != ""))
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
	smartCostLimit := func(lp loadpoint.API, limit *float64) {
		lp.SetSmartCostLimit(limit)
	}
	smartFeedInPriorityLimit := func(lp loadpoint.API, limit *float64) {
		lp.SetSmartFeedInPriorityLimit(limit)
	}

	routes := map[string]route{
		"health":                  {"GET", "/health", healthHandler(site)},
		"buffersoc":               {"POST", "/buffersoc/{value:[0-9.]+}", floatHandler(site.SetBufferSoc, site.GetBufferSoc)},
		"bufferstartsoc":          {"POST", "/bufferstartsoc/{value:[0-9.]+}", floatHandler(site.SetBufferStartSoc, site.GetBufferStartSoc)},
		"batterydischargecontrol": {"POST", "/batterydischargecontrol/{value:[01truefalse]+}", boolHandler(site.SetBatteryDischargeControl, site.GetBatteryDischargeControl)},
		"batterygridcharge":       {"POST", "/batterygridchargelimit/{value:-?[0-9.]+}", floatPtrHandler(site.SetBatteryGridChargeLimit, site.GetBatteryGridChargeLimit)},
		"batterygridchargedelete": {"DELETE", "/batterygridchargelimit", floatPtrHandler(site.SetBatteryGridChargeLimit, site.GetBatteryGridChargeLimit)},
		"batterymode":             {"POST", "/batterymode/{value:[a-z]+}", updateBatteryMode(site)},
		"batterymodedelete":       {"DELETE", "/batterymode", updateBatteryMode(site)},
		"prioritysoc":             {"POST", "/prioritysoc/{value:[0-9.]+}", floatHandler(site.SetPrioritySoc, site.GetPrioritySoc)},
		"residualpower":           {"POST", "/residualpower/{value:-?[0-9.]+}", floatHandler(site.SetResidualPower, site.GetResidualPower)},
		"smartcost":               {"POST", "/smartcostlimit/{value:-?[0-9.]+}", updateSmartCostLimit(site, smartCostLimit)},
		"smartcostdelete":         {"DELETE", "/smartcostlimit", updateSmartCostLimit(site, smartCostLimit)},
		"smartfeedin":             {"POST", "/smartfeedinprioritylimit/{value:-?[0-9.]+}", updateSmartCostLimit(site, smartFeedInPriorityLimit)},
		"smartfeedindelete":       {"DELETE", "/smartfeedinprioritylimit", updateSmartCostLimit(site, smartFeedInPriorityLimit)},
		"tariff":                  {"GET", "/tariff/{tariff:[a-z]+}", tariffHandler(site)},
		"sessions":                {"GET", "/sessions", sessionHandler},
		"updatesession":           {"PUT", "/session/{id:[0-9]+}", updateSessionHandler},
		"deletesession":           {"DELETE", "/session/{id:[0-9]+}", deleteSessionHandler},
		"telemetry2":              {"POST", "/settings/telemetry/{value:[01truefalse]+}", boolHandler(telemetry.Enable, telemetry.Enabled)},
	}

	for _, r := range routes {
		api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
	}

	// vehicle api
	vehicles := map[string]route{
		"minsoc":         {"POST", "/vehicles/{name:[a-zA-Z0-9_.:-]+}/minsoc/{value:[0-9]+}", minSocHandler(site)},
		"limitsoc":       {"POST", "/vehicles/{name:[a-zA-Z0-9_.:-]+}/limitsoc/{value:[0-9]+}", limitSocHandler(site)},
		"plan":           {"POST", "/vehicles/{name:[a-zA-Z0-9_.:-]+}/plan/soc/{value:[0-9]+}/{time:[0-9TZ:.+-]+}", planSocHandler(site)},
		"plan2":          {"DELETE", "/vehicles/{name:[a-zA-Z0-9_.:-]+}/plan/soc", planSocRemoveHandler(site)},
		"repeatingPlans": {"POST", "/vehicles/{name:[a-zA-Z0-9_.:-]+}/plan/repeating", addRepeatingPlansHandler(site)},

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
	// TODO any loadpoint
	for id, lp := range site.Loadpoints() {
		api := api.PathPrefix(fmt.Sprintf("/loadpoints/%d", id+1)).Subrouter()

		routes := map[string]route{
			"mode":                      {"POST", "/mode/{value:[a-z]+}", handler(eapi.ChargeModeString, pass(lp.SetMode), lp.GetMode)},
			"limitsoc":                  {"POST", "/limitsoc/{value:[0-9]+}", intHandler(pass(lp.SetLimitSoc), lp.GetLimitSoc)},
			"limitenergy":               {"POST", "/limitenergy/{value:[0-9.]+}", floatHandler(pass(lp.SetLimitEnergy), lp.GetLimitEnergy)},
			"mincurrent":                {"POST", "/mincurrent/{value:[0-9.]+}", floatHandler(lp.SetMinCurrent, lp.GetMinCurrent)},
			"maxcurrent":                {"POST", "/maxcurrent/{value:[0-9.]+}", floatHandler(lp.SetMaxCurrent, lp.GetMaxCurrent)},
			"phases":                    {"POST", "/phases/{value:[0-9]+}", intHandler(lp.SetPhasesConfigured, lp.GetPhasesConfigured)},
			"plan":                      {"GET", "/plan", planHandler(lp)},
			"staticPlanPreview":         {"GET", "/plan/static/preview/{type:(?:soc|energy)}/{value:[0-9.]+}/{time:[0-9TZ:.+-]+}", staticPlanPreviewHandler(lp)},
			"repeatingPlanPreview":      {"GET", "/plan/repeating/preview/{soc:[0-9]+}/{weekdays:[0-6,]+}/{time:[0-2][0-9]:[0-5][0-9]}/{tz:[a-zA-Z0-9_./:-]+}", repeatingPlanPreviewHandler(lp)},
			"planenergy":                {"POST", "/plan/energy/{value:[0-9.]+}/{time:[0-9TZ:.+-]+}", planEnergyHandler(lp)},
			"planenergy2":               {"DELETE", "/plan/energy", planRemoveHandler(lp)},
			"vehicle":                   {"POST", "/vehicle/{name:[a-zA-Z0-9_.:-]+}", vehicleSelectHandler(site, lp)},
			"vehicle2":                  {"DELETE", "/vehicle", vehicleRemoveHandler(lp)},
			"vehicleDetect":             {"PATCH", "/vehicle", vehicleDetectHandler(lp)},
			"enableThreshold":           {"POST", "/enable/threshold/{value:-?[0-9.]+}", floatHandler(pass(lp.SetEnableThreshold), lp.GetEnableThreshold)},
			"enableDelay":               {"POST", "/enable/delay/{value:[0-9]+}", durationHandler(pass(lp.SetEnableDelay), lp.GetEnableDelay)},
			"disableThreshold":          {"POST", "/disable/threshold/{value:-?[0-9.]+}", floatHandler(pass(lp.SetDisableThreshold), lp.GetDisableThreshold)},
			"disableDelay":              {"POST", "/disable/delay/{value:[0-9]+}", durationHandler(pass(lp.SetDisableDelay), lp.GetDisableDelay)},
			"smartCost":                 {"POST", "/smartcostlimit/{value:-?[0-9.]+}", floatPtrHandler(pass(lp.SetSmartCostLimit), lp.GetSmartCostLimit)},
			"smartCostDelete":           {"DELETE", "/smartcostlimit", floatPtrHandler(pass(lp.SetSmartCostLimit), lp.GetSmartCostLimit)},
			"smartFeedInPriority":       {"POST", "/smartfeedinprioritylimit/{value:-?[0-9.]+}", floatPtrHandler(pass(lp.SetSmartFeedInPriorityLimit), lp.GetSmartFeedInPriorityLimit)},
			"smartFeedInPriorityDelete": {"DELETE", "/smartfeedinprioritylimit", floatPtrHandler(pass(lp.SetSmartFeedInPriorityLimit), lp.GetSmartFeedInPriorityLimit)},
			"priority":                  {"POST", "/priority/{value:[0-9]+}", intHandler(pass(lp.SetPriority), lp.GetPriority)},
			"batteryBoost":              {"POST", "/batteryboost/{value:[01truefalse]+}", boolHandler(lp.SetBatteryBoost, func() bool { return lp.GetBatteryBoost() > 0 })},
		}

		for _, r := range routes {
			api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
		}
	}
}

// RegisterSystemHandler provides system level handlers
func (s *HTTPd) RegisterSystemHandler(site *core.Site, valueChan chan<- util.Param, cache *util.ParamCache, auth auth.Auth, shutdown func(), configFile string) {
	router := s.Server.Handler.(*mux.Router)

	// api
	api := router.PathPrefix("/api").Subrouter()
	api.Use(jsonHandler)
	api.Use(handlers.CompressHandler)
	api.Use(handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type"}),
	))

	if site == nil {
		// If site is nil, create a new empty site. Settings will be loaded during this process and
		// site meter references and title can be updated using APIs.
		var err error
		site, err = core.NewSiteFromConfig(nil)
		if err != nil {
			// should not happen
			panic(err)
		}
	}

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
			"devices":            {"GET", "/devices/{class:[a-z]+}", devicesConfigHandler},
			"device":             {"GET", "/devices/{class:[a-z]+}/{id:[0-9.]+}", deviceConfigHandler},
			"devicestatus":       {"GET", "/devices/{class:[a-z]+}/{name:[a-zA-Z0-9_.:-]+}/status", deviceStatusHandler},
			"dirty":              {"GET", "/dirty", getHandler(ConfigDirty)},
			"evccyaml":           {"GET", "/evcc.yaml", configYamlHandler(configFile)},
			"newdevice":          {"POST", "/devices/{class:[a-z]+}", newDeviceHandler},
			"updatedevice":       {"PUT", "/devices/{class:[a-z]+}/{id:[0-9.]+}", updateDeviceHandler},
			"deletedevice":       {"DELETE", "/devices/{class:[a-z]+}/{id:[0-9.]+}", deleteDeviceHandler(site)},
			"testconfig":         {"POST", "/test/{class:[a-z]+}", testConfigHandler},
			"testmerged":         {"POST", "/test/{class:[a-z]+}/merge/{id:[0-9.]+}", testConfigHandler},
			"interval":           {"POST", "/interval/{value:[0-9.]+}", settingsSetDurationHandler(keys.Interval)},
			"updatesponsortoken": {"POST", "/sponsortoken", updateSponsortokenHandler},
			"deletesponsortoken": {"DELETE", "/sponsortoken", deleteSponsorTokenHandler},
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
			keys.Shm:     func() any { return new(shm.Config) },
			keys.Influx:  func() any { return new(globalconfig.Influx) },
		} {
			routes["update"+key] = route{Method: "POST", Pattern: "/" + key, HandlerFunc: settingsSetJsonHandler(key, valueChan, fun)}
			routes["delete"+key] = route{Method: "DELETE", Pattern: "/" + key, HandlerFunc: settingsDeleteJsonHandler(key, valueChan, fun())}
		}

		for _, r := range routes {
			api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
		}

		// site
		for _, r := range map[string]route{
			"site":       {"GET", "/site", siteHandler(site)},
			"updatesite": {"PUT", "/site", updateSiteHandler(site)},
		} {
			api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
		}

		// loadpoints
		for _, r := range map[string]route{
			"loadpoints":      {"GET", "/loadpoints", loadpointsConfigHandler()},
			"loadpoint":       {"GET", "/loadpoints/{id:[0-9.]+}", loadpointConfigHandler()},
			"updateloadpoint": {"PUT", "/loadpoints/{id:[0-9.]+}", updateLoadpointHandler()},
			"deleteloadpoint": {"DELETE", "/loadpoints/{id:[0-9.]+}", deleteLoadpointHandler()},
			"newloadpoint":    {"POST", "/loadpoints", newLoadpointHandler()},
		} {
			api.Methods(r.Methods()...).Path(r.Pattern).Handler(r.HandlerFunc)
		}
	}

	{ // api/system
		api := api.PathPrefix("/system").Subrouter()
		api.Use(ensureAuthHandler(auth))

		// system api
		routes := map[string]route{
			"log":        {"GET", "/log", logHandler},
			"logareas":   {"GET", "/log/areas", logAreasHandler},
			"clearcache": {"DELETE", "/cache", clearCacheHandler},
			"backup":     {"POST", "/backup", getBackup(auth)},
			"restore":    {"POST", "/restore", restoreDatabase(auth, shutdown)},
			"reset":      {"POST", "/reset", resetDatabase(auth, shutdown)},
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
