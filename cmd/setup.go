package cmd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	paho "github.com/eclipse/paho.mqtt.golang"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/charger"
	"github.com/evcc-io/evcc/charger/eebus"
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/core/keys"
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
	"github.com/evcc-io/evcc/server/modbus"
	"github.com/evcc-io/evcc/server/oauth2redirect"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/locale"
	"github.com/evcc-io/evcc/util/machine"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/evcc-io/evcc/vehicle"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/libp2p/zeroconf/v2"
	"github.com/spf13/cast"
	"github.com/spf13/cobra"
	"golang.org/x/sync/errgroup"
	"golang.org/x/text/currency"
)

var conf = globalconfig.All{
	Interval: 10 * time.Second,
	Log:      "info",
	Network: globalconfig.Network{
		Schema: "http",
		Host:   "evcc.local",
		Port:   7070,
	},
	Mqtt: globalconfig.Mqtt{
		Topic: "evcc",
	},
	Database: globalconfig.DB{
		Type: "sqlite",
		Dsn:  "~/.evcc/evcc.db",
	},
}

var nameRE = regexp.MustCompile(`^[a-zA-Z0-9_.:-]+$`)

func nameValid(name string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf("name must not contain special characters or spaces: %s", name)
	}
	return nil
}

func tokenDanger(conf []config.Named) bool {
	problematic := []string{"tesla", "psa", "opel", "citroen", "ds", "peugeot"}

	for _, cc := range conf {
		if slices.Contains(problematic, cc.Type) {
			return true
		}
		template, ok := cc.Other["template"].(string)
		if ok && cc.Type == "template" && slices.Contains(problematic, template) {
			return true
		}
	}

	return false
}

func loadConfigFile(conf *globalconfig.All, checkDB bool) error {
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

	// check service database
	if _, err := os.Stat(serviceDB); err == nil && checkDB && conf.Database.Dsn != serviceDB && tokenDanger(conf.Vehicles) {
		log.FATAL.Fatal(`

Found systemd service database at "` + serviceDB + `", evcc has been invoked with database "` + conf.Database.Dsn + `".
Running evcc with vehicles configured in evcc.yaml may lead to expiring the yaml configuration's vehicle tokens.
This is due to the fact, that the token refresh will be saved to the local instead of the service's database.
If you have vehicles with touchy tokens like PSA or Tesla, make sure to remove vehicle configuration from the yaml file.

If you know what you're doing, you can run evcc ignoring the service database with the --ignore-db flag.
`)
	}

	// parse log levels after reading config
	if err == nil {
		parseLogLevels()
	}

	return err
}

func configureCircuits(static []config.Named, names ...string) error {
	children := slices.Clone(static)

	// TODO: check for circular references
NEXT:
	for i, cc := range children {
		if cc.Name == "" {
			return fmt.Errorf("cannot create circuit: missing name")
		}

		if err := nameValid(cc.Name); err != nil {
			return fmt.Errorf("cannot create circuit: duplicate name: %s", cc.Name)
		}

		if parent := cast.ToString(cc.Property("parent")); parent != "" {
			if _, err := config.Circuits().ByName(parent); err != nil {
				continue
			}
		}

		log := util.NewLogger("circuit-" + cc.Name)
		instance, err := core.NewCircuitFromConfig(log, cc.Other)
		if err != nil {
			return fmt.Errorf("cannot create circuit '%s': %w", cc.Name, err)
		}

		// ensure config has title
		if instance.GetTitle() == "" {
			//lint:ignore SA1019 as Title is safe on ascii
			instance.SetTitle(strings.Title(cc.Name))
		}

		if err := config.Circuits().Add(config.NewStaticDevice(cc, instance)); err != nil {
			return err
		}

		children = slices.Delete(children, i, i+1)
		goto NEXT
	}

	if len(children) > 0 {
		return fmt.Errorf("circuit is missing parent: %s", children[0].Name)
	}

	// append devices from database
	configurable, err := config.ConfigurationsByClass(templates.Circuit)
	if err != nil {
		return err
	}

	children2 := slices.Clone(configurable)

NEXT2:
	for i, conf := range children2 {
		cc := conf.Named()

		if len(names) > 0 && !slices.Contains(names, cc.Name) {
			return nil
		}

		if parent := cast.ToString(cc.Property("parent")); parent != "" {
			if _, err := config.Circuits().ByName(parent); err != nil {
				continue
			}
		}

		log := util.NewLogger("circuit-" + cc.Name)
		instance, err := core.NewCircuitFromConfig(log, cc.Other)
		if err != nil {
			return fmt.Errorf("cannot create circuit '%s': %w", cc.Name, err)
		}

		// ensure config has title
		if instance.GetTitle() == "" {
			//lint:ignore SA1019 as Title is safe on ascii
			instance.SetTitle(strings.Title(cc.Name))
		}

		if err := config.Circuits().Add(config.NewConfigurableDevice(conf, instance)); err != nil {
			return err
		}

		children2 = slices.Delete(children2, i, i+1)
		goto NEXT2
	}

	if len(children2) > 0 {
		return fmt.Errorf("missing parent circuit: %s", children2[0].Named().Name)
	}

	var rootFound bool
	for _, dev := range config.Circuits().Devices() {
		c := dev.Instance()

		if c.GetParent() == nil {
			if rootFound {
				return errors.New("cannot have multiple root circuits")
			}
			rootFound = true
		}
	}

	if !rootFound && len(config.Circuits().Devices()) > 0 {
		return errors.New("root circuit required")
	}

	return nil
}

func configureMeters(static []config.Named, names ...string) error {
	for i, cc := range static {
		if cc.Name == "" {
			return fmt.Errorf("cannot create meter %d: missing name", i+1)
		}

		if len(names) > 0 && !slices.Contains(names, cc.Name) {
			continue
		}

		if err := nameValid(cc.Name); err != nil {
			log.WARN.Printf("create meter %d: %v", i+1, err)
		}

		instance, err := meter.NewFromConfig(cc.Type, cc.Other)
		if err != nil {
			return &DeviceError{cc.Name, fmt.Errorf("cannot create meter '%s': %w", cc.Name, err)}
		}

		if err := config.Meters().Add(config.NewStaticDevice(cc, instance)); err != nil {
			return &DeviceError{cc.Name, err}
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

		// TOTO add fake devices

		instance, err := meter.NewFromConfig(cc.Type, cc.Other)
		if err != nil {
			return &DeviceError{cc.Name, fmt.Errorf("cannot create meter '%s': %w", cc.Name, err)}
		}

		if err := config.Meters().Add(config.NewConfigurableDevice(conf, instance)); err != nil {
			return &DeviceError{cc.Name, err}
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

		if err := nameValid(cc.Name); err != nil {
			log.WARN.Printf("create charger %d: %v", i+1, err)
		}

		g.Go(func() error {
			instance, err := charger.NewFromConfig(cc.Type, cc.Other)
			if err != nil {
				return &DeviceError{cc.Name, fmt.Errorf("cannot create charger '%s': %w", cc.Name, err)}
			}

			if err := config.Chargers().Add(config.NewStaticDevice(cc, instance)); err != nil {
				return &DeviceError{cc.Name, err}
			}

			return nil
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

			// TOTO add fake devices

			instance, err := charger.NewFromConfig(cc.Type, cc.Other)
			if err != nil {
				return fmt.Errorf("cannot create charger '%s': %w", cc.Name, err)
			}

			if err := config.Chargers().Add(config.NewConfigurableDevice(conf, instance)); err != nil {
				return &DeviceError{cc.Name, err}
			}

			return nil
		})
	}

	return g.Wait()
}

func vehicleInstance(cc config.Named) (api.Vehicle, error) {
	instance, err := vehicle.NewFromConfig(cc.Type, cc.Other)
	if err != nil {
		var ce *util.ConfigError
		if errors.As(err, &ce) {
			return nil, err
		}

		// wrap non-config vehicle errors to prevent fatals
		log.ERROR.Printf("creating vehicle %s failed: %v", cc.Name, err)
		instance = vehicle.NewWrapper(cc.Name, cc.Type, cc.Other, err)
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

		if err := nameValid(cc.Name); err != nil {
			return fmt.Errorf("cannot create vehicle %d: %w", i+1, err)
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

func configureSponsorship(token string) (err error) {
	if settings.Exists(keys.SponsorToken) {
		if token, err = settings.String(keys.SponsorToken); err != nil {
			return err
		}
	}

	// TODO migrate settings

	return sponsor.ConfigureSponsorship(token)
}

func configureEnvironment(cmd *cobra.Command, conf *globalconfig.All) (err error) {
	// full http request log
	if cmd.Flags().Lookup(flagHeaders).Changed {
		request.LogHeaders = true
	}

	// setup machine id
	if conf.Plant != "" {
		// TODO decide wrapping
		err = machine.CustomID(conf.Plant)
	}

	// setup sponsorship (allow env override)
	if err == nil {
		err = wrapErrorWithClass(ClassSponsorship, configureSponsorship(conf.SponsorToken))
	}

	// setup translations
	if err == nil {
		// TODO decide wrapping
		err = locale.Init()
	}

	// setup persistence
	if err == nil && conf.Database.Dsn != "" {
		err = wrapErrorWithClass(ClassDatabase, configureDatabase(conf.Database))
	}

	// setup mqtt client listener
	if err == nil {
		err = wrapErrorWithClass(ClassMqtt, configureMqtt(&conf.Mqtt))
	}

	// setup EEBus server
	if err == nil {
		err = wrapErrorWithClass(ClassEEBus, configureEEBus(conf.EEBus))
	}

	// setup javascript VMs
	if err == nil {
		err = wrapErrorWithClass(ClassJavascript, configureJavascript(conf.Javascript))
	}

	// setup go VMs
	if err == nil {
		err = wrapErrorWithClass(ClassGo, configureGo(conf.Go))
	}

	// setup config database
	if err == nil {
		// TODO decide wrapping
		err = config.Init(db.Instance)
	}

	return
}

// configureDatabase configures session database
func configureDatabase(conf globalconfig.DB) error {
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
func configureInflux(conf *globalconfig.Influx) (*server.Influx, error) {
	// read settings
	if settings.Exists(keys.Influx) {
		if err := settings.Json(keys.Influx, &conf); err != nil {
			return nil, err
		}
	}

	if conf.URL == "" {
		return nil, nil
	}

	// TODO remove yaml file
	// // migrate settings
	// if !settings.Exists(keys.Influx) {
	// 	if err := settings.SetJson(keys.Influx, conf); err != nil {
	// 		return nil, err
	// 	}
	// }

	influx := server.NewInfluxClient(
		conf.URL,
		conf.Token,
		conf.Org,
		conf.User,
		conf.Password,
		conf.Database,
	)

	return influx, nil
}

// setup mqtt
func configureMqtt(conf *globalconfig.Mqtt) error {
	// migrate settings
	if settings.Exists(keys.Mqtt) {
		if err := settings.Json(keys.Mqtt, &conf); err != nil {
			return err
		}

		// TODO remove yaml file
		// } else {
		// 	// migrate settings & write defaults
		// 	if err := settings.SetJson(keys.Mqtt, conf); err != nil {
		// 		return err
		// 	}
	}

	if conf.Broker == "" {
		return nil
	}

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
func configureJavascript(conf []globalconfig.Javascript) error {
	for _, cc := range conf {
		if _, err := javascript.RegisteredVM(cc.VM, cc.Script); err != nil {
			return fmt.Errorf("failed configuring javascript: %w", err)
		}
	}
	return nil
}

// setup go
func configureGo(conf []globalconfig.Go) error {
	for _, cc := range conf {
		if _, err := golang.RegisteredVM(cc.VM, cc.Script); err != nil {
			return fmt.Errorf("failed configuring go: %w", err)
		}
	}
	return nil
}

// setup HEMS
func configureHEMS(conf config.Typed, site *core.Site, httpd *server.HTTPd) error {
	// migrate settings
	if settings.Exists(keys.Hems) {
		if err := settings.Yaml(keys.Hems, new(map[string]any), &conf); err != nil {
			return err
		}
	}

	if conf.Type == "" {
		return nil
	}

	// TODO remove yaml file
	// // migrate settings
	// if !settings.Exists(keys.Hems) {
	// 	if err := settings.SetYaml(keys.Hems, conf); err != nil {
	// 		return err
	// 	}
	// }

	hems, err := hems.NewFromConfig(conf.Type, conf.Other, site, httpd)
	if err != nil {
		return fmt.Errorf("failed configuring hems: %w", err)
	}

	go hems.Run()

	return nil
}

// networkSettings reads/migrates network settings
func networkSettings(conf *globalconfig.Network) error {
	if settings.Exists(keys.Network) {
		return settings.Json(keys.Network, &conf)
	}

	// TODO remove yaml file
	// // migrate settings
	// return settings.SetJson(keys.Network, conf)

	return nil
}

// setup MDNS
func configureMDNS(conf globalconfig.Network) error {
	host := strings.TrimSuffix(conf.Host, ".local")

	zc, err := zeroconf.RegisterProxy("evcc", "_http._tcp", "local.", conf.Port, host, nil, []string{"path=/"}, nil)
	if err != nil {
		return fmt.Errorf("mDNS announcement: %w", err)
	}

	shutdown.Register(zc.Shutdown)

	return nil
}

// setup EEBus
func configureEEBus(conf eebus.Config) error {
	// migrate settings
	if settings.Exists(keys.EEBus) {
		if err := settings.Yaml(keys.EEBus, new(map[string]any), &conf); err != nil {
			return err
		}
	}

	if conf.URI == "" {
		return nil
	}

	// TODO remove yaml file
	// // migrate settings
	// if !settings.Exists(keys.EEBus) {
	// 	if err := settings.SetYaml(keys.EEBus, conf); err != nil {
	// 		return err
	// 	}
	// }

	var err error
	if eebus.Instance, err = eebus.NewServer(conf); err != nil {
		return fmt.Errorf("failed configuring eebus: %w", err)
	}

	eebus.Instance.Run()
	shutdown.Register(eebus.Instance.Shutdown)

	return nil
}

// setup messaging
func configureMessengers(conf globalconfig.Messaging, vehicles push.Vehicles, valueChan chan<- util.Param, cache *util.Cache) (chan push.Event, error) {
	// migrate settings
	if settings.Exists(keys.Messaging) {
		if err := settings.Yaml(keys.Messaging, new(map[string]any), &conf); err != nil {
			return nil, err
		}

		// TODO remove yaml file
		// } else if len(conf.Services)+len(conf.Events) > 0 {
		// 	if err := settings.SetYaml(keys.Messaging, conf); err != nil {
		// 		return nil, err
		// 	}
	}

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

func configureTariff(name string, conf config.Typed, t *api.Tariff) error {
	if conf.Type == "" {
		return nil
	}

	res, err := tariff.NewFromConfig(conf.Type, conf.Other)
	if err != nil {
		return &DeviceError{name, err}
	}

	*t = res
	return nil
}

func configureTariffs(conf globalconfig.Tariffs) (*tariff.Tariffs, error) {
	// migrate settings
	if settings.Exists(keys.Tariffs) {
		if err := settings.Yaml(keys.Tariffs, new(map[string]any), &conf); err != nil {
			return nil, err
		}

		// TODO remove yaml file
		// } else if conf.Grid.Type != "" || conf.FeedIn.Type != "" || conf.Co2.Type != "" || conf.Planner.Type != "" {
		// 	if err := settings.SetYaml(keys.Tariffs, conf); err != nil {
		// 		return nil, err
		// 	}
	}

	tariffs := tariff.Tariffs{
		Currency: currency.EUR,
	}

	if conf.Currency != "" {
		tariffs.Currency = currency.MustParseISO(conf.Currency)
	}

	g, _ := errgroup.WithContext(context.Background())
	g.Go(func() error { return configureTariff("grid", conf.Grid, &tariffs.Grid) })
	g.Go(func() error { return configureTariff("feedin", conf.FeedIn, &tariffs.FeedIn) })
	g.Go(func() error { return configureTariff("co2", conf.Co2, &tariffs.Co2) })
	g.Go(func() error { return configureTariff("planner", conf.Planner, &tariffs.Planner) })

	if err := g.Wait(); err != nil {
		return nil, &ClassError{ClassTariff, err}
	}

	return &tariffs, nil
}

func configureDevices(conf globalconfig.All) error {
	// TODO: add name/identifier to error for better highlighting in UI
	if err := configureMeters(conf.Meters); err != nil {
		return &ClassError{ClassMeter, err}
	}
	if err := configureChargers(conf.Chargers); err != nil {
		return &ClassError{ClassCharger, err}
	}
	if err := configureVehicles(conf.Vehicles); err != nil {
		return &ClassError{ClassVehicle, err}
	}
	if err := configureCircuits(conf.Circuits); err != nil {
		return &ClassError{ClassCircuit, err}
	}
	return nil
}

func configureModbusProxy(conf []globalconfig.ModbusProxy) error {
	// migrate settings
	if settings.Exists(keys.ModbusProxy) {
		if err := settings.Yaml(keys.ModbusProxy, new([]map[string]any), &conf); err != nil {
			return err
		}

		// TODO remove yaml file
		// } else if len(conf) > 0 {
		// 	if err := settings.SetYaml(keys.ModbusProxy, conf); err != nil {
		// 		return err
		// 	}
	}

	for _, cfg := range conf {
		var mode modbus.ReadOnlyMode
		mode, err := modbus.ReadOnlyModeString(cfg.ReadOnly)
		if err != nil {
			return err
		}

		if err = modbus.StartProxy(cfg.Port, cfg.Settings, mode); err != nil {
			return err
		}
	}

	return nil
}

func configureSiteAndLoadpoints(conf *globalconfig.All) (*core.Site, error) {
	// migrate settings
	if settings.Exists(keys.Interval) {
		d, err := settings.Int(keys.Interval)
		if err != nil {
			return nil, err
		}
		conf.Interval = time.Duration(d)

		// TODO remove yaml file
		// } else if conf.Interval != 0 {
		// settings.SetInt(keys.Interval, int64(conf.Interval))
	}

	if err := configureDevices(*conf); err != nil {
		return nil, err
	}

	loadpoints, err := configureLoadpoints(*conf)
	if err != nil {
		return nil, fmt.Errorf("failed configuring loadpoints: %w", err)
	}

	tariffs, err := configureTariffs(conf.Tariffs)
	if err != nil {
		return nil, &ClassError{ClassTariff, err}
	}

	site, err := configureSite(conf.Site, loadpoints, tariffs)
	if err != nil {
		return nil, err
	}

	if len(config.Circuits().Devices()) > 0 {
		if err := validateCircuits(site, loadpoints); err != nil {
			return nil, &ClassError{ClassCircuit, err}
		}
	}

	return site, nil
}

func validateCircuits(site site.API, loadpoints []*core.Loadpoint) error {
CONTINUE:
	for _, dev := range config.Circuits().Devices() {
		instance := dev.Instance()

		if instance.HasMeter() || site.GetCircuit() == instance {
			continue
		}

		for _, lp := range loadpoints {
			if lp.GetCircuit() == instance {
				continue CONTINUE
			}
		}

		return fmt.Errorf("circuit %s has no meter or loadpoint assigned", dev.Config().Name)
	}

	if site.GetCircuit() == nil {
		return errors.New("site has no circuit")
	}

	return nil
}

func configureSite(conf map[string]interface{}, loadpoints []*core.Loadpoint, tariffs *tariff.Tariffs) (*core.Site, error) {
	site, err := core.NewSiteFromConfig(conf)
	if err != nil {
		return nil, err
	}

	if err := site.Boot(log, loadpoints, tariffs); err != nil {
		return nil, fmt.Errorf("failed configuring site: %w", err)
	}

	if len(config.Circuits().Devices()) > 0 && site.GetCircuit() == nil {
		return nil, errors.New("site has no circuit")
	}

	return site, nil
}

func configureLoadpoints(conf globalconfig.All) ([]*core.Loadpoint, error) {
	if len(conf.Loadpoints) == 0 {
		return nil, errors.New("missing loadpoints")
	}

	var loadpoints []*core.Loadpoint

	for id, cfg := range conf.Loadpoints {
		log := util.NewLoggerWithLoadpoint("lp-"+strconv.Itoa(id+1), id+1)
		settings := &core.Settings{Key: "lp" + strconv.Itoa(id+1) + "."}

		lp, err := core.NewLoadpointFromConfig(log, settings, cfg)
		if err != nil {
			return nil, fmt.Errorf("failed configuring loadpoint: %w", err)
		}

		loadpoints = append(loadpoints, lp)
	}

	return loadpoints, nil
}

// configureAuth handles routing for devices. For now only api.AuthProvider related routes
func configureAuth(conf globalconfig.Network, vehicles []api.Vehicle, router *mux.Router, paramC chan<- util.Param) {
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
