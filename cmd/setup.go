package cmd

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/andig/evcc/core"
	"github.com/andig/evcc/hems"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/push"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/pipe"
	"github.com/spf13/viper"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// setup influx databases
func configureDatabase(conf server.InfluxConfig, loadPoints []*core.LoadPoint, in <-chan util.Param) {
	influx := server.NewInfluxClient(
		conf.URL,
		conf.Token,
		conf.Org,
		conf.User,
		conf.Password,
		conf.Database,
	)

	// eliminate duplicate values
	dedupe := pipe.NewDeduplicator(30*time.Minute, "socCharge")
	in = dedupe.Pipe(in)

	// reduce number of values written to influx
	limiter := pipe.NewLimiter(5 * time.Second)
	in = limiter.Pipe(in)

	go influx.Run(loadPoints, in)
}

func mqttClientID() string {
	pid := rand.Int31()
	return fmt.Sprintf("evcc-%d", pid)
}

// setup mqtt
func configureMQTT(conf provider.MqttConfig) {
	provider.MQTT = provider.NewMqttClient(conf.Broker, conf.User, conf.Password, mqttClientID(), 1)
}

// setup HEMS
func configureHEMS(conf string, site *core.Site, cache *util.Cache, httpd *server.HTTPd) hems.HEMS {
	hems, err := hems.NewFromConfig(conf, site, cache, httpd)
	if err != nil {
		log.FATAL.Fatalf("failed configuring hems: %v", err)
	}
	return hems
}

// setup messaging
func configureMessengers(conf messagingConfig, cache *util.Cache) chan push.Event {
	notificationChan := make(chan push.Event, 1)
	notificationHub := push.NewHub(conf.Events, cache)

	for _, service := range conf.Services {
		impl, err := push.NewMessengerFromConfig(service.Type, service.Other)
		if err != nil {
			log.FATAL.Fatal(err)
			log.FATAL.Fatalf("failed configuring messenger %s: %v", service.Type, err)
		}
		notificationHub.Add(impl)
	}

	go notificationHub.Run(notificationChan)

	return notificationChan
}

func loadConfig(conf config) *core.Site {
	cp := &ConfigProvider{}
	cp.configure(conf)

	loadPoints := configureLoadPoints(conf, cp)
	site := configureSite(conf.Site, cp, loadPoints)

	return site
}

func configureSite(conf map[string]interface{}, cp *ConfigProvider, loadPoints []*core.LoadPoint) *core.Site {
	site, err := core.NewSiteFromConfig(log, cp, conf, loadPoints)
	if err != nil {
		log.FATAL.Fatalf("failed configuring site: %v", err)
	}

	return site
}

func configureLoadPoints(conf config, cp *ConfigProvider) (loadPoints []*core.LoadPoint) {
	lpInterfaces, ok := viper.AllSettings()["loadpoints"].([]interface{})
	if !ok || len(lpInterfaces) == 0 {
		log.FATAL.Fatal("missing loadpoints")
	}

	for id, lpcI := range lpInterfaces {
		var lpc map[string]interface{}
		if err := util.DecodeOther(lpcI, &lpc); err != nil {
			log.FATAL.Fatalf("failed decoding loadpoint configuration: %v", err)
		}

		log := util.NewLogger("lp-" + strconv.Itoa(id+1))
		lp, err := core.NewLoadPointFromConfig(log, cp, lpc)
		if err != nil {
			log.FATAL.Fatalf("failed configuring loadpoint: %v", err)
		}

		loadPoints = append(loadPoints, lp)
	}

	return loadPoints
}

func loadConfigFile(cfgFile string) (conf config) {
	if cfgFile != "" {
		log.INFO.Println("using config file", cfgFile)
		if err := viper.UnmarshalExact(&conf); err != nil {
			log.FATAL.Fatalf("failed parsing config file %s: %v", cfgFile, err)
		}
	} else {
		log.FATAL.Fatal("missing evcc config")
	}

	return conf
}
