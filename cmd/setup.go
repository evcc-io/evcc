package cmd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/charger/eebus"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/core/site"
	"github.com/evcc-io/evcc/hems"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/provider/golang"
	"github.com/evcc-io/evcc/provider/javascript"
	"github.com/evcc-io/evcc/provider/mqtt"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/server/oauth2redirect"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/locale"
	"github.com/evcc-io/evcc/util/machine"
	"github.com/evcc-io/evcc/util/modbus"
	"github.com/evcc-io/evcc/util/pipe"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/libp2p/zeroconf/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/currency"
)

var conf = globalConfig{
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

var nameRE = regexp.MustCompile(`^[a-zA-Z0-9_.:-]+$`)

type globalConfig struct {
	Network      networkConfig
	Log          string
	SponsorToken string
	Plant        string // telemetry plant id
	Telemetry    bool
	Metrics      bool
	Profile      bool
	Levels       map[string]string
	Interval     time.Duration
	Database     dbConfig
	Mqtt         mqttConfig
	ModbusProxy  []proxyConfig
	Javascript   []javascriptConfig
	Go           []goConfig
	Influx       server.InfluxConfig
	EEBus        map[string]interface{}
	HEMS         config.Typed
	Messaging    messagingConfig
	Meters       []config.Named
	Chargers     []config.Named
	Vehicles     []config.Named
	Tariffs      tariffConfig
	Site         map[string]interface{}
	Loadpoints   []map[string]interface{}
}

type mqttConfig struct {
	mqtt.Config `mapstructure:",squash"`
	Topic       string
}

type javascriptConfig struct {
	VM     string
	Script string
}

type goConfig struct {
	VM     string
	Script string
}

type proxyConfig struct {
	Port            int
	ReadOnly        string
	modbus.Settings `mapstructure:",squash"`
}

type dbConfig struct {
	Type string
	Dsn  string
}

type messagingConfig struct {
	Events   map[string]push.EventTemplateConfig
	Services []config.Typed
}

type tariffConfig struct {
	Currency string
	Grid     config.Typed
	FeedIn   config.Typed
	Co2      config.Typed
	Planner  config.Typed
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

func loadConfigFile(conf *globalConfig) error {
	err := viper.ReadInConfig()

	if cfgFile = viper.ConfigFileUsed(); cfgFile == "" {
		return err
	}

	log.INFO.Println("using config file:", cfgFile)

	if err == nil {
		if err = viper.UnmarshalExact(conf); err != nil {
			err = fmt.Errorf("failed parsing config file: %w", err)
		}
	}

	// parse log levels after reading config
	if err == nil {
		parseLogLevels()
	}

	return err
}

func configureMeters(static []config.Named, names ...string) error {
	for i, cc := range static {
		if cc.Name == "" {
			return fmt.Errorf("cannot create meter %d: missing name", i+1)
		}

		if len(names) > 0 && !slices.Contains(names, cc.Name) {
			continue
		}

		instance, err := meter.NewFromConfig(cc.Type, cc.Other)
		if err != nil {
			return fmt.Errorf("cannot create meter '%s': %w", cc.Name, err)
		}

		if err := config.Meters().Add(config.NewStaticDevice(cc, instance)); err != nil {
			return err
		}
	}

	// append devices from database
	configurable, err := config.ConfigurationsByClass(templates.Meter)
	if err != nil {
		return err
	}

	for _, conf := range configurable {
		cc := conf.Named()

		if len(names) > 0 && !slices.Contains(names, cc.Name) {
			return nil
		}

		instance, err := meter.NewFromConfig(cc.Type, cc.Other)
		if err != nil {
			return fmt.Errorf("cannot create meter '%s': %w", cc.Name, err)
		}

		if err := config.Meters().Add(config.NewConfigurableDevice(conf, instance)); err != nil {
			return err
		}
	}

	return nil
}

func configureChargers(static []config.Named, names ...string) error {
	g, _ := errgroup.WithContext(context.Background())

	for i, cc := range static {
		if cc.Name == "" {
			return fmt.Errorf("cannot create charger %d: missing name", i+1)
		}

		if len(names) > 0 && !slices.Contains(names, cc.Name) {
			continue
		}

		g.Go(func() error {
			instance, err := charger.NewFromConfig(cc.Type, cc.Other)
			if err != nil {
				return fmt.Errorf("cannot create charger '%s': %w", cc.Name, err)
			}

			return config.Chargers().Add(config.NewStaticDevice(cc, instance))
		})
	}

	// append devices from database
	configurable, err := config.ConfigurationsByClass(templates.Charger)
	if err != nil {
		return err
	}

	for _, conf := range configurable {
		g.Go(func() error {
			cc := conf.Named()

			if len(names) > 0 && !slices.Contains(names, cc.Name) {
				return nil
			}

			instance, err := charger.NewFromConfig(cc.Type, cc.Other)
			if err != nil {
				return fmt.Errorf("cannot create charger '%s': %w", cc.Name, err)
			}

			return config.Chargers().Add(config.NewConfigurableDevice(conf, instance))
		})
	}

	return g.Wait()
}

func vehicleInstance(cc config.Named) (api.Vehicle, error) {
	if !nameRE.MatchString(cc.Name) {
		return nil, fmt.Errorf("vehicle name must not contain special characters or spaces: %s", cc.Name)
	}

	instance, err := vehicle.NewFromConfig(cc.Type, cc.Other)
	if err != nil {
		var ce *util.ConfigError
		if errors.As(err, &ce) {
			return nil, err
		}

		// wrap non-config vehicle errors to prevent fatals
		log.ERROR.Printf("creating vehicle %s failed: %v", cc.Name, err)
		instance = vehicle.NewWrapper(cc.Name, cc.Other, err)
	}

	// ensure vehicle config has title
	if instance.Title() == "" {
		//lint:ignore SA1019 as Title is safe on ascii
		instance.SetTitle(strings.Title(cc.Name))
	}

	return instance, nil
}

func configureVehicles(static []config.Named, names ...string) error {
	var mu sync.Mutex
	g, _ := errgroup.WithContext(context.Background())

	// stable-sort vehicles by name
	devs1 := make([]config.Device[api.Vehicle], 0, len(static))

	for i, cc := range static {
		if cc.Name == "" {
			return fmt.Errorf("cannot create vehicle %d: missing name", i+1)
		}

		if len(names) > 0 && !slices.Contains(names, cc.Name) {
			continue
		}

		g.Go(func() error {
			instance, err := vehicleInstance(cc)
			if err != nil {
				return fmt.Errorf("cannot create vehicle '%s': %w", cc.Name, err)
			}

			mu.Lock()
			defer mu.Unlock()
			devs1 = append(devs1, config.NewStaticDevice(cc, instance))

			return nil
		})
	}

	// append devices from database
	configurable, err := config.ConfigurationsByClass(templates.Vehicle)
	if err != nil {
		return err
	}

	// stable-sort vehicles by id
	devs2 := make([]config.ConfigurableDevice[api.Vehicle], 0, len(configurable))

	for _, conf := range configurable {
		g.Go(func() error {
			cc := conf.Named()

			if len(names) > 0 && !slices.Contains(names, cc.Name) {
				return nil
			}

			instance, err := vehicleInstance(cc)
			if err != nil {
				return fmt.Errorf("cannot create vehicle '%s': %w", cc.Name, err)
			}

			mu.Lock()
			defer mu.Unlock()
			devs2 = append(devs2, config.NewConfigurableDevice(conf, instance))

			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	slices.SortFunc(devs1, func(i, j config.Device[api.Vehicle]) int {
		return cmp.Compare(strings.ToLower(i.Config().Name), strings.ToLower(j.Config().Name))
	})

	for _, dev := range devs1 {
		if err := config.Vehicles().Add(dev); err != nil {
			return err
		}
	}

	slices.SortFunc(devs2, func(i, j config.ConfigurableDevice[api.Vehicle]) int {
		return cmp.Compare(i.ID(), j.ID())
	})

	for _, dev := range devs2 {
		if err := config.Vehicles().Add(dev); err != nil {
			return err
		}
	}

	return nil
}

func configureEnvironment(cmd *cobra.Command, conf globalConfig) (err error) {
	// full http request log
	if cmd.Flags().Lookup(flagHeaders).Changed {
		request.LogHeaders = true
	}

	// setup machine id
	if conf.Plant != "" {
		err = machine.CustomID(conf.Plant)
	}

	// setup sponsorship (allow env override)
	if err == nil {
		err = sponsor.ConfigureSponsorship(conf.SponsorToken)
	}

	// setup translations
	if err == nil {
		err = locale.Init()
	}

	// setup persistence
	if err == nil && conf.Database.Dsn != "" {
		err = configureDatabase(conf.Database)
	}

	// setup mqtt client listener
	if err == nil && conf.Mqtt.Broker != "" {
		err = configureMQTT(conf.Mqtt)
	}

	// setup javascript VMs
	if err == nil {
		err = configureJavascript(conf.Javascript)
	}

	// setup go VMs
	if err == nil {
		err = configureGo(conf.Go)
	}

	// setup EEBus server
	if err == nil && conf.EEBus != nil {
		err = configureEEBus(conf.EEBus)
	}

	// setup config database
	if err == nil {
		err = config.Init(db.Instance)
	}

	return
}

// configureDatabase configures session database
func configureDatabase(conf dbConfig) error {
	if err := db.NewInstance(conf.Type, conf.Dsn); err != nil {
		return err
	}

	if err := settings.Init(); err != nil {
		return err
	}

	persistSettings := func() {
		if err := settings.Persist(); err != nil {
			log.ERROR.Println("cannot save settings:", err)
		}
	}

	// persist unsaved settings on shutdown
	shutdown.Register(persistSettings)

	// persist unsaved settings every 30 minutes
	go func() {
		for range time.Tick(30 * time.Minute) {
			persistSettings()
		}
	}()

	return nil
}

// configureInflux configures influx database
func configureInflux(conf server.InfluxConfig, site site.API, in <-chan util.Param) {
	influx := server.NewInfluxClient(
		conf.URL,
		conf.Token,
		conf.Org,
		conf.User,
		conf.Password,
		conf.Database,
	)

	// eliminate duplicate values
	dedupe := pipe.NewDeduplicator(30*time.Minute, "vehicleCapacity", "vehicleSoc", "vehicleRange", "vehicleOdometer", "chargedEnergy", "chargeRemainingEnergy")
	in = dedupe.Pipe(in)

	go influx.Run(site, in)
}

// setup mqtt
func configureMQTT(conf mqttConfig) error {
	log := util.NewLogger("mqtt")

	instance, err := mqtt.RegisteredClient(log, conf.Broker, conf.User, conf.Password, conf.ClientID, 1, conf.Insecure, func(options *paho.ClientOptions) {
		topic := fmt.Sprintf("%s/status", strings.Trim(conf.Topic, "/"))
		options.SetWill(topic, "offline", 1, true)

		oc := options.OnConnect
		options.SetOnConnectHandler(func(client paho.Client) {
			oc(client)                                   // original handler
			_ = client.Publish(topic, 1, true, "online") // alive - not logged
		})
	})
	if err != nil {
		return fmt.Errorf("failed configuring mqtt: %w", err)
	}

	mqtt.Instance = instance
	return nil
}

// setup javascript
func configureJavascript(conf []javascriptConfig) error {
	for _, cc := range conf {
		if _, err := javascript.RegisteredVM(cc.VM, cc.Script); err != nil {
			return fmt.Errorf("failed configuring javascript: %w", err)
		}
	}
	return nil
}

// setup go
func configureGo(conf []goConfig) error {
	for _, cc := range conf {
		if _, err := golang.RegisteredVM(cc.VM, cc.Script); err != nil {
			return fmt.Errorf("failed configuring go: %w", err)
		}
	}
	return nil
}

// setup HEMS
func configureHEMS(conf config.Typed, site *core.Site, httpd *server.HTTPd) error {
	hems, err := hems.NewFromConfig(conf.Type, conf.Other, site, httpd)
	if err != nil {
		return fmt.Errorf("failed configuring hems: %w", err)
	}

	go hems.Run()

	return nil
}

// setup MDNS
func configureMDNS(conf networkConfig) error {
	host := strings.TrimSuffix(conf.Host, ".local")

	zc, err := zeroconf.RegisterProxy("evcc", "_http._tcp", "local.", conf.Port, host, nil, []string{"path=/"}, nil)
	if err != nil {
		return fmt.Errorf("mDNS announcement: %w", err)
	}

	shutdown.Register(zc.Shutdown)

	return nil
}

// setup EEBus
func configureEEBus(conf map[string]interface{}) error {
	var err error
	if eebus.Instance, err = eebus.NewServer(conf); err != nil {
		return fmt.Errorf("failed configuring eebus: %w", err)
	}

	eebus.Instance.Run()
	shutdown.Register(eebus.Instance.Shutdown)

	return nil
}

// setup messaging
func configureMessengers(conf messagingConfig, vehicles push.Vehicles, valueChan chan util.Param, cache *util.Cache) (chan push.Event, error) {
	messageChan := make(chan push.Event, 1)

	messageHub, err := push.NewHub(conf.Events, vehicles, cache)
	if err != nil {
		return messageChan, fmt.Errorf("failed configuring push services: %w", err)
	}

	for _, service := range conf.Services {
		impl, err := push.NewFromConfig(service.Type, service.Other)
		if err != nil {
			return messageChan, fmt.Errorf("failed configuring push service %s: %w", service.Type, err)
		}
		messageHub.Add(impl)
	}

	go messageHub.Run(messageChan, valueChan)

	return messageChan, nil
}

func configureTariff(name string, conf config.Typed, t *api.Tariff, wg *sync.WaitGroup) {
	defer wg.Done()

	if conf.Type == "" {
		return
	}

	res, err := tariff.NewFromConfig(conf.Type, conf.Other)
	if err != nil {
		log.ERROR.Printf("failed configuring %s tariff: %v", name, err)
		return
	}

	*t = res
}

func configureTariffs(conf tariffConfig) (*tariff.Tariffs, error) {
	tariffs := tariff.Tariffs{
		Currency: currency.EUR,
	}

	if conf.Currency != "" {
		tariffs.Currency = currency.MustParseISO(conf.Currency)
	}

	var wg sync.WaitGroup
	wg.Add(4)

	go configureTariff("grid", conf.Grid, &tariffs.Grid, &wg)
	go configureTariff("feedin", conf.FeedIn, &tariffs.FeedIn, &wg)
	go configureTariff("co2", conf.Co2, &tariffs.Co2, &wg)
	go configureTariff("planner", conf.Planner, &tariffs.Planner, &wg)

	wg.Wait()

	return &tariffs, nil
}

func configureDevices(conf globalConfig) error {
	if err := configureMeters(conf.Meters); err != nil {
		return err
	}
	if err := configureChargers(conf.Chargers); err != nil {
		return err
	}
	return configureVehicles(conf.Vehicles)
}

func configureSiteAndLoadpoints(conf globalConfig) (*core.Site, error) {
	if err := configureDevices(conf); err != nil {
		return nil, err
	}

	loadpoints, err := configureLoadpoints(conf)
	if err != nil {
		return nil, fmt.Errorf("failed configuring loadpoints: %w", err)
	}

	tariffs, err := configureTariffs(conf.Tariffs)
	if err != nil {
		return nil, err
	}

	return configureSite(conf.Site, loadpoints, tariffs)
}

func configureSite(conf map[string]interface{}, loadpoints []*core.Loadpoint, tariffs *tariff.Tariffs) (*core.Site, error) {
	site, err := core.NewSiteFromConfig(log, conf, loadpoints, tariffs)
	if err != nil {
		return nil, fmt.Errorf("failed configuring site: %w", err)
	}

	return site, nil
}

func configureLoadpoints(conf globalConfig) (loadpoints []*core.Loadpoint, err error) {
	if len(conf.Loadpoints) == 0 {
		return nil, errors.New("missing loadpoints")
	}

	for id, lpc := range conf.Loadpoints {
		log := util.NewLoggerWithLoadpoint("lp-"+strconv.Itoa(id+1), id+1)
		settings := &core.Settings{Key: "lp" + strconv.Itoa(id+1) + "."}

		lp, err := core.NewLoadpointFromConfig(log, settings, lpc)
		if err != nil {
			return nil, fmt.Errorf("failed configuring loadpoint: %w", err)
		}

		loadpoints = append(loadpoints, lp)
	}

	return loadpoints, nil
}

// configureAuth handles routing for devices. For now only api.AuthProvider related routes
func configureAuth(conf networkConfig, vehicles []api.Vehicle, router *mux.Router, paramC chan<- util.Param) {
	auth := router.PathPrefix("/oauth").Subrouter()
	auth.Use(handlers.CompressHandler)
	auth.Use(handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type"}),
	))

	// wire the handler
	oauth2redirect.SetupRouter(auth)

	// initialize
	authCollection := util.NewAuthCollection(paramC)

	baseURI := conf.URI()
	baseAuthURI := fmt.Sprintf("%s/oauth", baseURI)

	var id int
	for _, v := range vehicles {
		if provider, ok := v.(api.AuthProvider); ok {
			id += 1

			basePath := fmt.Sprintf("vehicles/%d", id)
			callbackURI := fmt.Sprintf("%s/%s/callback", baseAuthURI, basePath)

			// register vehicle
			ap := authCollection.Register(fmt.Sprintf("oauth/%s", basePath), v.Title())

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

	authCollection.Publish()
}
