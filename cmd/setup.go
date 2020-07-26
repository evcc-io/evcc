package cmd

import (
	"fmt"
	"math/rand"
	"strconv"
	"time"

	"github.com/andig/evcc/core"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/push"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
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
	dedupe := server.NewDeduplicator(30*time.Minute, "socCharge")
	in = dedupe.Pipe(in)

	// reduce number of values written to influx
	limiter := server.NewLimiter(5 * time.Second)
	in = limiter.Pipe(in)

	go influx.Run(loadPoints, in)
}

func mqttClientID() string {
	pid := rand.Int31()
	return fmt.Sprintf("evcc-%d", pid)
}

// setup mqtt
func configureMQTT(conf provider.MqttConfig, site *core.Site, in <-chan util.Param) {
	provider.MQTT = provider.NewMqttClient(conf.Broker, conf.User, conf.Password, mqttClientID(), 1)

	if site != nil && conf.Topic != "" {
		mqtt := &server.MQTT{Handler: provider.MQTT}
		go mqtt.Run(conf.Topic, site, in)
	}
}

func configureMessengers(conf messagingConfig, cache *util.Cache) chan push.Event {
	notificationChan := make(chan push.Event, 1)
	notificationHub := push.NewHub(conf.Events, cache)

	for _, service := range conf.Services {
		impl := push.NewMessengerFromConfig(service.Type, service.Other)
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
	return core.NewSiteFromConfig(log, cp, conf, loadPoints)
}

func configureLoadPoints(conf config, cp *ConfigProvider) (loadPoints []*core.LoadPoint) {
	// slice of loadpoints
	lps, ok := viper.AllSettings()["loadpoints"]
	if !ok {
		log.FATAL.Fatal("config: missing loadpoints")
	}

	// decode slice into slice of maps
	var lpc []map[string]interface{}
	util.DecodeOther(log, lps, &lpc)

	for id, lpc := range lpc {
		log := util.NewLogger("lp-" + strconv.Itoa(id+1))
		lp := core.NewLoadPointFromConfig(log, cp, lpc)
		loadPoints = append(loadPoints, lp)
	}

	return
}

func loadConfigFile(cfgFile string) (conf config) {
	if cfgFile != "" {
		log.INFO.Println("using config file", cfgFile)
		if err := viper.UnmarshalExact(&conf); err != nil {
			log.FATAL.Fatalf("config: failed parsing config file %s: %v", cfgFile, err)
		}
	} else {
		log.FATAL.Fatal("missing evcc config")
	}
	return
}
