package cmd

import (
	"fmt"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/server"
	autoauth "github.com/evcc-io/evcc/server/auth"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/evcc-io/evcc/vehicle/wrapper"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"golang.org/x/exp/maps"
)

var conf = config{
	Interval: 10 * time.Second,
	Log:      "info",
	Network: networkConfig{
		Schema: "http",
		Host:   "evcc.local",
		Port:   7070,
	},
	Mqtt: mqttConfig{
		Topic: "evcc",
	},
	Database: dbConfig{
		Type: "sqlite",
		Dsn:  "~/.evcc/evcc.db",
	},
}

type config struct {
	URI          interface{} // TODO deprecated
	Network      networkConfig
	Log          string
	SponsorToken string
	Telemetry    bool
	Plant        string // telemetry plant id
	Metrics      bool
	Profile      bool
	Levels       map[string]string
	Interval     time.Duration
	Mqtt         mqttConfig
	Database     dbConfig
	Javascript   map[string]interface{}
	Influx       server.InfluxConfig
	EEBus        map[string]interface{}
	HEMS         typedConfig
	Messaging    messagingConfig
	Meters       []qualifiedConfig
	Chargers     []qualifiedConfig
	Vehicles     []qualifiedConfig
	Tariffs      tariffConfig
	Site         map[string]interface{}
	LoadPoints   []map[string]interface{}
}

type mqttConfig struct {
	mqtt.Config `mapstructure:",squash"`
	Topic       string
}

type dbConfig struct {
	Type string
	Dsn  string
}

type qualifiedConfig struct {
	Name, Type string
	Other      map[string]interface{} `mapstructure:",remain"`
}

type typedConfig struct {
	Type  string
	Other map[string]interface{} `mapstructure:",remain"`
}

type messagingConfig struct {
	Events   map[string]push.EventTemplateConfig
	Services []typedConfig
}

type tariffConfig struct {
	Currency string
	Grid     typedConfig
	FeedIn   typedConfig
}

type networkConfig struct {
	Schema string
	Host   string
	Port   int
}

func (c networkConfig) HostPort() string {
	if c.Schema == "http" && c.Port == 80 || c.Schema == "https" && c.Port == 443 {
		return c.Host
	}
	return net.JoinHostPort(c.Host, strconv.Itoa(c.Port))
}

func (c networkConfig) URI() string {
	return fmt.Sprintf("%s://%s", c.Schema, c.HostPort())
}

// ConfigProvider provides configuration items
type ConfigProvider struct {
	meters   map[string]api.Meter
	chargers map[string]api.Charger
	vehicles map[string]api.Vehicle
	visited  map[string]bool
	auth     *util.AuthCollection
}

func (cp *ConfigProvider) TrackVisitors() {
	cp.visited = make(map[string]bool)
}

// Meter provides meters by name
func (cp *ConfigProvider) Meter(name string) (api.Meter, error) {
	if meter, ok := cp.meters[name]; ok {
		// track duplicate usage https://github.com/evcc-io/evcc/issues/1744
		if cp.visited != nil {
			if _, ok := cp.visited[name]; ok {
				log.FATAL.Fatalf("duplicate meter usage: %s", name)
			}
			cp.visited[name] = true
		}

		return meter, nil
	}
	return nil, fmt.Errorf("meter does not exist: %s", name)
}

// Charger provides chargers by name
func (cp *ConfigProvider) Charger(name string) (api.Charger, error) {
	if charger, ok := cp.chargers[name]; ok {
		return charger, nil
	}
	return nil, fmt.Errorf("charger does not exist: %s", name)
}

// Vehicle provides vehicles by name
func (cp *ConfigProvider) Vehicle(name string) (api.Vehicle, error) {
	if vehicle, ok := cp.vehicles[name]; ok {
		return vehicle, nil
	}
	return nil, fmt.Errorf("vehicle does not exist: %s", name)
}

func (cp *ConfigProvider) configure(conf config) error {
	err := cp.configureMeters(conf)
	if err == nil {
		err = cp.configureChargers(conf)
	}
	if err == nil {
		err = cp.configureVehicles(conf)
	}
	return err
}

func (cp *ConfigProvider) configureMeters(conf config) error {
	cp.meters = make(map[string]api.Meter)
	for id, cc := range conf.Meters {
		if cc.Name == "" {
			return fmt.Errorf("cannot create %s meter: missing name", humanize.Ordinal(id+1))
		}

		m, err := meter.NewFromConfig(cc.Type, cc.Other)
		if err != nil {
			err = fmt.Errorf("cannot create meter '%s': %w", cc.Name, err)
			return err
		}

		if _, exists := cp.meters[cc.Name]; exists {
			return fmt.Errorf("duplicate meter name: %s already defined and must be unique", cc.Name)
		}

		cp.meters[cc.Name] = m
	}

	return nil
}

func (cp *ConfigProvider) configureChargers(conf config) error {
	cp.chargers = make(map[string]api.Charger)
	for id, cc := range conf.Chargers {
		if cc.Name == "" {
			return fmt.Errorf("cannot create %s charger: missing name", humanize.Ordinal(id+1))
		}

		c, err := charger.NewFromConfig(cc.Type, cc.Other)
		if err != nil {
			err = fmt.Errorf("cannot create charger '%s': %w", cc.Name, err)
			return err
		}

		if _, exists := cp.chargers[cc.Name]; exists {
			return fmt.Errorf("duplicate charger name: %s already defined and must be unique", cc.Name)
		}

		cp.chargers[cc.Name] = c
	}

	return nil
}

func (cp *ConfigProvider) configureVehicles(conf config) error {
	cp.vehicles = make(map[string]api.Vehicle)
	for id, cc := range conf.Vehicles {
		if cc.Name == "" {
			return fmt.Errorf("cannot create %s vehicle: missing name", humanize.Ordinal(id+1))
		}

		// ensure vehicle config has title
		var ccWithTitle struct {
			Title string
			Other map[string]interface{} `mapstructure:",remain"`
		}

		if err := util.DecodeOther(cc.Other, &ccWithTitle); err != nil {
			return err
		}

		if ccWithTitle.Title == "" {
			//lint:ignore SA1019 as Title is safe on ascii
			cc.Other["title"] = strings.Title(cc.Name)
		}

		v, err := vehicle.NewFromConfig(cc.Type, cc.Other)
		if err != nil {
			// wrap any created errors to prevent fatals
			v, _ = wrapper.New(v, err)
		}

		if _, exists := cp.vehicles[cc.Name]; exists {
			return fmt.Errorf("duplicate vehicle name: %s already defined and must be unique", cc.Name)
		}

		cp.vehicles[cc.Name] = v
	}

	return nil
}

// webControl handles routing for devices. For now only api.AuthProvider related routes
func (cp *ConfigProvider) webControl(conf networkConfig, router *mux.Router, paramC chan<- util.Param) {
	auth := router.PathPrefix("/oauth").Subrouter()
	auth.Use(handlers.CompressHandler)
	auth.Use(handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type"}),
	))

	// wire the handler
	autoauth.Setup(auth)

	// initialize
	cp.auth = util.NewAuthCollection(paramC)

	baseURI := conf.URI()
	baseAuthURI := fmt.Sprintf("%s/oauth", baseURI)

	// stable map iteration
	keys := maps.Keys(cp.vehicles)
	sort.Strings(keys)

	var id int
	for _, k := range keys {
		v := cp.vehicles[k]

		if provider, ok := v.(api.AuthProvider); ok {
			id += 1

			basePath := fmt.Sprintf("vehicles/%d", id)
			callbackURI := fmt.Sprintf("%s/%s/callback", baseAuthURI, basePath)

			// register vehicle
			ap := cp.auth.Register(fmt.Sprintf("oauth/%s", basePath), v.Title())

			provider.SetCallbackParams(baseURI, callbackURI, ap.Handler())

			auth.
				Methods(http.MethodPost).
				Path(fmt.Sprintf("/%s/login", basePath)).
				HandlerFunc(provider.LoginHandler())
			auth.
				Methods(http.MethodPost).
				Path(fmt.Sprintf("/%s/logout", basePath)).
				HandlerFunc(provider.LogoutHandler())

			log.INFO.Printf("ensure the oauth client redirect/callback is configured for %s: %s", v.Title(), callbackURI)
		}
	}

	cp.auth.Publish()
}
