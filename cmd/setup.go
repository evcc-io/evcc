package cmd

import (
	"cmp"
	"context"
	"errors"
	"fmt"
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
	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/core/circuit"
	"github.com/evcc-io/evcc/core/keys"
	"github.com/evcc-io/evcc/core/loadpoint"
	coresettings "github.com/evcc-io/evcc/core/settings"
	"github.com/evcc-io/evcc/hems"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/plugin/golang"
	"github.com/evcc-io/evcc/plugin/javascript"
	"github.com/evcc-io/evcc/plugin/mqtt"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/server/db"
	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/evcc-io/evcc/server/eebus"
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
	"github.com/samber/lo"
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
	EEBus: eebus.Config{
		URI: ":4712",
	},
	Database: globalconfig.DB{
		Type: "sqlite",
		Dsn:  "",
	},
}

var nameRE = regexp.MustCompile(`^[a-zA-Z0-9_.:-]+$`)

func nameValid(name string) error {
	if !nameRE.MatchString(name) {
		return fmt.Errorf("name must not contain special characters or spaces: %s", name)
	}
	return nil
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

	// user did not specify a database path
	if conf.Database.Dsn == "" && checkDB {
		// check if service database exists
		if _, err := os.Stat(serviceDB); err == nil {
			// service database found, ask user what to do
			sudo := ""
			if !isWritable(serviceDB) {
				sudo = "sudo "
			}
			log.FATAL.Fatal(`
Found systemd service database at "` + serviceDB + `", evcc has been invoked with no explicit database path.
Running the same config with multiple databases can lead to expiring vehicle tokens.

If you want to use the existing service database run the following command:

` + sudo + `evcc --database ` + serviceDB + `

If you want to create a new user-space database run the following command:

evcc --database ~/.evcc/evcc.db

If you know what you're doing, you can skip the database check with the --ignore-db flag.
			`)
		}
	}

	// parse log levels after reading config
	if err == nil {
		parseLogLevels()
	}

	return err
}

func isWritable(filePath string) bool {
	file, err := os.OpenFile(filePath, os.O_WRONLY, 0o666)
	if err != nil {
		return false
	}
	file.Close()
	return true
}

func configureCircuits(conf *[]config.Named) error {
	// migrate settings
	if settings.Exists(keys.Circuits) {
		*conf = []config.Named{}
		if err := settings.Yaml(keys.Circuits, new([]map[string]any), &conf); err != nil {
			return err
		}
	}

	children := slices.Clone(*conf)

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

		props, err := customDevice(cc.Other)
		if err != nil {
			return fmt.Errorf("cannot decode custom circuit '%s': %w", cc.Name, err)
		}

		instance, err := circuit.NewFromConfig(context.TODO(), log, props)
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
	var eg errgroup.Group

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

		eg.Go(func() error {
			ctx := util.WithLogger(context.TODO(), util.NewLogger(cc.Name))

			instance, err := meter.NewFromConfig(ctx, cc.Type, cc.Other)
			if err != nil {
				return &DeviceError{cc.Name, fmt.Errorf("cannot create meter '%s': %w", cc.Name, err)}
			}

			if err := config.Meters().Add(config.NewStaticDevice(cc, instance)); err != nil {
				return &DeviceError{cc.Name, err}
			}

			return nil
		})
	}

	// append devices from database
	configurable, err := config.ConfigurationsByClass(templates.Meter)
	if err != nil {
		return err
	}

	for _, conf := range configurable {
		eg.Go(func() error {
			cc := conf.Named()

			if len(names) > 0 && !slices.Contains(names, cc.Name) {
				return nil
			}

			ctx := util.WithLogger(context.TODO(), util.NewLogger(cc.Name))

			props, err := customDevice(cc.Other)
			if err != nil {
				err = &DeviceError{cc.Name, fmt.Errorf("cannot decode custom meter '%s': %w", cc.Name, err)}
			}

			var instance api.Meter
			if err == nil {
				instance, err = meter.NewFromConfig(ctx, cc.Type, props)
				if err != nil {
					err = &DeviceError{cc.Name, fmt.Errorf("cannot create meter '%s': %w", cc.Name, err)}
				}
			}

			if e := config.Meters().Add(config.NewConfigurableDevice(&conf, instance)); e != nil && err == nil {
				err = &DeviceError{cc.Name, e}
			}

			return err
		})
	}

	return eg.Wait()
}

func configureChargers(static []config.Named, names ...string) error {
	var eg errgroup.Group

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

		eg.Go(func() error {
			ctx := util.WithLogger(context.TODO(), util.NewLogger(cc.Name))

			instance, err := charger.NewFromConfig(ctx, cc.Type, cc.Other)
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
		eg.Go(func() error {
			cc := conf.Named()

			if len(names) > 0 && !slices.Contains(names, cc.Name) {
				return nil
			}

			ctx := util.WithLogger(context.TODO(), util.NewLogger(cc.Name))

			props, err := customDevice(cc.Other)
			if err != nil {
				err = &DeviceError{cc.Name, fmt.Errorf("cannot decode custom charger '%s': %w", cc.Name, err)}
			}

			var instance api.Charger
			if err == nil {
				instance, err = charger.NewFromConfig(ctx, cc.Type, props)
				if err != nil {
					err = &DeviceError{cc.Name, fmt.Errorf("cannot create charger '%s': %w", cc.Name, err)}
				}
			}

			if e := config.Chargers().Add(config.NewConfigurableDevice(&conf, instance)); e != nil && err == nil {
				err = &DeviceError{cc.Name, e}
			}

			return err
		})
	}

	return eg.Wait()
}

func vehicleInstance(cc config.Named) (api.Vehicle, error) {
	ctx := util.WithLogger(context.TODO(), util.NewLogger(cc.Name))

	props, err := customDevice(cc.Other)

	var instance api.Vehicle
	if err == nil {
		instance, err = vehicle.NewFromConfig(ctx, cc.Type, props)
	}

	if err != nil {
		if ce := new(util.ConfigError); errors.As(err, &ce) {
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
	var eg errgroup.Group

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

		eg.Go(func() error {
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
		eg.Go(func() error {
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
			devs2 = append(devs2, config.NewConfigurableDevice(&conf, instance))

			return nil
		})
	}

	if err := eg.Wait(); err != nil {
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

func configureEnvironment(cmd *cobra.Command, conf *globalconfig.All) error {
	// full http request log
	if cmd.Flag(flagHeaders).Changed {
		request.LogHeaders = true
	}

	// setup persistence
	err := wrapErrorWithClass(ClassDatabase, configureDatabase(conf.Database))

	// setup additional templates
	if err == nil {
		if cmd.PersistentFlags().Changed(flagTemplate) {
			class, err := templates.ClassString(cmd.PersistentFlags().Lookup(flagTemplateType).Value.String())
			if err != nil {
				return err
			}

			if err := templates.Register(class, cmd.Flag(flagTemplate).Value.String()); err != nil {
				return err
			}
		}
	}

	// setup translations
	if err == nil {
		// TODO decide wrapping
		err = locale.Init()
	}

	// setup machine id
	if err == nil && conf.Plant != "" {
		// TODO decide wrapping
		err = machine.CustomID(conf.Plant)
	}

	// setup sponsorship (allow env override)
	if err == nil {
		err = wrapErrorWithClass(ClassSponsorship, configureSponsorship(conf.SponsorToken))
	}

	// setup mqtt client listener
	if err == nil {
		err = wrapErrorWithClass(ClassMqtt, configureMqtt(&conf.Mqtt))
	}

	// setup EEBus server
	if err == nil {
		err = wrapErrorWithClass(ClassEEBus, configureEEBus(&conf.EEBus))
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

	return err
}

// configureDatabase configures session database
func configureDatabase(conf globalconfig.DB) error {
	if conf.Dsn == "" {
		conf.Dsn = userDB
	}

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
		for range time.Tick(time.Minute) {
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

	influx := server.NewInfluxClient(
		conf.URL,
		conf.Token,
		conf.Org,
		conf.User,
		conf.Password,
		conf.Database,
		conf.Insecure,
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
	}

	if conf.Broker == "" {
		return nil
	}

	log := util.NewLogger("mqtt")

	instance, err := mqtt.RegisteredClient(log, conf.Broker, conf.User, conf.Password, conf.ClientID, 1, conf.Insecure, conf.CaCert, conf.ClientCert, conf.ClientKey, func(options *paho.ClientOptions) {
		if !runAsService || conf.Topic == "" {
			return
		}

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
func configureHEMS(conf *globalconfig.Hems, site *core.Site, httpd *server.HTTPd) error {
	// migrate settings
	if settings.Exists(keys.Hems) {
		*conf = globalconfig.Hems{}
		if err := settings.Yaml(keys.Hems, new(map[string]any), &conf); err != nil {
			return err
		}
	}

	if conf.Type == "" {
		return nil
	}

	props, err := customDevice(conf.Other)
	if err != nil {
		return fmt.Errorf("cannot decode custom hems '%s': %w", conf.Type, err)
	}

	hems, err := hems.NewFromConfig(context.TODO(), conf.Type, props, site, httpd)
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
func configureEEBus(conf *eebus.Config) error {
	// migrate settings
	if settings.Exists(keys.EEBus) {
		*conf = eebus.Config{}
		if err := settings.Yaml(keys.EEBus, new(map[string]any), &conf); err != nil {
			return err
		}
	}

	if !conf.Configured() {
		return nil
	}

	var err error
	if eebus.Instance, err = eebus.NewServer(*conf); err != nil {
		return fmt.Errorf("failed configuring eebus: %w", err)
	}

	eebus.Instance.Run()
	shutdown.Register(eebus.Instance.Shutdown)

	return nil
}

// setup messaging
func configureMessengers(conf *globalconfig.Messaging, vehicles push.Vehicles, valueChan chan<- util.Param, cache *util.ParamCache) (chan push.Event, error) {
	// migrate settings
	if settings.Exists(keys.Messaging) {
		*conf = globalconfig.Messaging{}
		if err := settings.Yaml(keys.Messaging, new(map[string]any), &conf); err != nil {
			return nil, err
		}
	}

	messageChan := make(chan push.Event, 1)

	messageHub, err := push.NewHub(conf.Events, vehicles, cache)
	if err != nil {
		return messageChan, fmt.Errorf("failed configuring push services: %w", err)
	}

	for _, conf := range conf.Services {
		props, err := customDevice(conf.Other)
		if err != nil {
			return nil, fmt.Errorf("cannot decode push service '%s': %w", conf.Type, err)
		}

		impl, err := push.NewFromConfig(context.TODO(), conf.Type, props)
		if err != nil {
			return messageChan, fmt.Errorf("failed configuring push service %s: %w", conf.Type, err)
		}
		messageHub.Add(impl)
	}

	go messageHub.Run(messageChan, valueChan)

	return messageChan, nil
}

func tariffInstance(name string, conf config.Typed) (api.Tariff, error) {
	ctx := util.WithLogger(context.TODO(), util.NewLogger(name))

	props, err := customDevice(conf.Other)
	if err != nil {
		return nil, fmt.Errorf("cannot decode custom tariff '%s': %w", name, err)
	}

	instance, err := tariff.NewFromConfig(ctx, conf.Type, props)
	if err != nil {
		if ce := new(util.ConfigError); errors.As(err, &ce) {
			return nil, err
		}

		// wrap non-config tariff errors to prevent fatals
		log.ERROR.Printf("creating tariff %s failed: %v", name, err)
		instance = tariff.NewWrapper(conf.Type, conf.Other, err)
	}

	return instance, nil
}

func configureTariff(u api.TariffUsage, conf config.Typed, t *api.Tariff) error {
	if conf.Type == "" {
		return nil
	}

	name := u.String()
	res, err := tariffInstance(name, conf)
	if err != nil {
		return &DeviceError{name, err}
	}

	*t = res
	return nil
}

func configureSolarTariff(conf []config.Typed, t *api.Tariff) error {
	var eg errgroup.Group
	tt := make([]api.Tariff, len(conf))

	for i, conf := range conf {
		eg.Go(func() error {
			if conf.Type == "" {
				return errors.New("missing type")
			}

			name := fmt.Sprintf("%s-%s-%d", api.TariffUsageSolar, tariff.Name(conf), i)
			res, err := tariffInstance(name, conf)
			if err != nil {
				return &DeviceError{name, err}
			}

			tt[i] = res
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	*t = tariff.NewCombined(tt)
	return nil
}

func configureTariffs(conf *globalconfig.Tariffs) (*tariff.Tariffs, error) {
	// migrate settings
	if settings.Exists(keys.Tariffs) {
		*conf = globalconfig.Tariffs{}
		if err := settings.Yaml(keys.Tariffs, new(map[string]any), &conf); err != nil {
			return nil, err
		}
	}

	tariffs := tariff.Tariffs{
		Currency: currency.EUR,
	}

	if conf.Currency != "" {
		tariffs.Currency = currency.MustParseISO(conf.Currency)
	}

	var eg errgroup.Group
	eg.Go(func() error { return configureTariff(api.TariffUsageGrid, conf.Grid, &tariffs.Grid) })
	eg.Go(func() error { return configureTariff(api.TariffUsageFeedIn, conf.FeedIn, &tariffs.FeedIn) })
	eg.Go(func() error { return configureTariff(api.TariffUsageCo2, conf.Co2, &tariffs.Co2) })
	eg.Go(func() error { return configureTariff(api.TariffUsagePlanner, conf.Planner, &tariffs.Planner) })
	if len(conf.Solar) == 1 {
		eg.Go(func() error { return configureTariff(api.TariffUsageSolar, conf.Solar[0], &tariffs.Solar) })
	} else {
		eg.Go(func() error { return configureSolarTariff(conf.Solar, &tariffs.Solar) })
	}

	if err := eg.Wait(); err != nil {
		return nil, &ClassError{ClassTariff, err}
	}

	return &tariffs, nil
}

func configureDevices(conf globalconfig.All) error {
	// collect references for filtering used devices
	if err := collectRefs(conf); err != nil {
		return err
	}

	// TODO: add name/identifier to error for better highlighting in UI
	if err := configureMeters(conf.Meters, references.meter...); err != nil {
		return &ClassError{ClassMeter, err}
	}
	if err := configureChargers(conf.Chargers, references.charger...); err != nil {
		return &ClassError{ClassCharger, err}
	}
	if err := configureVehicles(conf.Vehicles); err != nil {
		return &ClassError{ClassVehicle, err}
	}
	if err := configureCircuits(&conf.Circuits); err != nil {
		return &ClassError{ClassCircuit, err}
	}
	return nil
}

func configureModbusProxy(conf *[]globalconfig.ModbusProxy) error {
	// migrate settings
	if settings.Exists(keys.ModbusProxy) {
		*conf = []globalconfig.ModbusProxy{}
		if err := settings.Yaml(keys.ModbusProxy, new([]map[string]any), &conf); err != nil {
			return err
		}
	}

	for _, cfg := range *conf {
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

	if err := configureLoadpoints(*conf); err != nil {
		return nil, &ClassError{ClassLoadpoint, err}
	}

	tariffs, err := configureTariffs(&conf.Tariffs)
	if err != nil {
		return nil, &ClassError{ClassTariff, err}
	}

	loadpoints := lo.Map(config.Loadpoints().Devices(), func(dev config.Device[loadpoint.API], _ int) *core.Loadpoint {
		lp := dev.Instance()
		return lp.(*core.Loadpoint)
	})

	site, err := configureSite(conf.Site, loadpoints, tariffs)
	if err != nil {
		return nil, err
	}

	if len(config.Circuits().Devices()) > 0 {
		if err := validateCircuits(loadpoints); err != nil {
			return nil, &ClassError{ClassCircuit, err}
		}
	}

	return site, nil
}

func validateCircuits(loadpoints []*core.Loadpoint) error {
	var hasRoot bool

CONTINUE:
	for _, dev := range config.Circuits().Devices() {
		instance := dev.Instance()

		isRoot := instance.GetParent() == nil
		if isRoot {
			if hasRoot {
				return errors.New("multiple root circuits")
			}

			hasRoot = true
		}

		if slices.ContainsFunc(loadpoints, func(lp *core.Loadpoint) bool {
			return lp.GetCircuit() == instance
		}) {
			continue CONTINUE
		}

		if !isRoot && !instance.HasMeter() {
			log.INFO.Printf("circuit %s has no meter and no loadpoint assigned", dev.Config().Name)
		}
	}

	if !hasRoot {
		return errors.New("missing root circuit")
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

	return site, nil
}

func configureLoadpoints(conf globalconfig.All) error {
	for id, cc := range conf.Loadpoints {
		cc.Name = "lp-" + strconv.Itoa(id+1)

		log := util.NewLoggerWithLoadpoint(cc.Name, id+1)
		settings := coresettings.NewDatabaseSettingsAdapter(fmt.Sprintf("lp%d.", id+1))

		instance, err := core.NewLoadpointFromConfig(log, settings, cc.Other)
		if err != nil {
			return &DeviceError{cc.Name, err}
		}

		if err := config.Loadpoints().Add(config.NewStaticDevice(cc, loadpoint.API(instance))); err != nil {
			return &DeviceError{cc.Name, err}
		}
	}

	// append devices from database
	configurable, err := config.ConfigurationsByClass(templates.Loadpoint)
	if err != nil {
		return err
	}

	for _, conf := range configurable {
		cc := conf.Named()

		id := len(config.Loadpoints().Devices())
		name := "lp-" + strconv.Itoa(id+1)
		log := util.NewLoggerWithLoadpoint(name, id+1)

		settings := coresettings.NewConfigSettingsAdapter(log, &conf)

		dynamic, static, err := loadpoint.SplitConfig(cc.Other)
		if err != nil {
			return &DeviceError{cc.Name, err}
		}

		instance, err := core.NewLoadpointFromConfig(log, settings, static)
		if err != nil {
			err = &DeviceError{cc.Name, err}
		}

		dev := config.NewConfigurableDevice[loadpoint.API](&conf, instance)
		if e := config.Loadpoints().Add(dev); e != nil && err == nil {
			err = &DeviceError{cc.Name, e}
		}

		if e := dynamic.Apply(instance); e != nil && err == nil {
			err = &DeviceError{cc.Name, e}
		}

		if err != nil {
			return err
		}
	}

	return nil
}

// configureAuth handles routing for devices. For now only api.AuthProvider related routes
func configureAuth(router *mux.Router) {
	auth := router.PathPrefix("/oauth").Subrouter()
	auth.Use(handlers.CompressHandler)
	auth.Use(handlers.CORS(
		handlers.AllowedHeaders([]string{"Content-Type"}),
	))

	// wire the handler
	oauth2redirect.SetupRouter(auth)
}
