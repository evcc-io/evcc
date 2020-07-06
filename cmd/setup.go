package cmd

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/andig/evcc/core"
	"github.com/andig/evcc/push"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/spf13/viper"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// setup influx databases
func configureDatabase(in <-chan util.Param, conf server.InfluxConfig) {
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

	go influx.Run(in)
}

func configureMessengers(conf messagingConfig) chan push.Event {
	notificationChan := make(chan push.Event, 1)
	notificationHub := push.NewHub(conf.Events)

	for _, service := range conf.Services {
		impl := push.NewMessengerFromConfig(service.Type, service.Other)
		notificationHub.Add(impl)
	}

	go notificationHub.Run(notificationChan)

	return notificationChan
}

func clientID() string {
	pid := rand.Int31()
	return fmt.Sprintf("evcc-%d", pid)
}

func loadConfig(conf config, eventsChan chan push.Event) (loadPoints []*core.LoadPoint) {
	cp := &ConfigProvider{}
	cp.configure(conf)

	// slice of loadpoints
	lps, ok := viper.AllSettings()["loadpoints"]
	if !ok {
		log.FATAL.Fatal("config: missing loadpoints")
	}

	// decode slice into slice of maps
	var lpc []map[string]interface{}
	util.DecodeOther(log, lps, &lpc)

	for _, lpc := range lpc {
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
