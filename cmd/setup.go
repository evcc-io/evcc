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

func dbTee(valueChan *chan core.Param) <-chan core.Param {
	var teeChan <-chan core.Param
	*valueChan, teeChan = tee(*valueChan)

	// eliminate duplicate values
	dedupe := server.NewDeduplicator(30*time.Minute, "socCharge")
	teeChan = dedupe.Pipe(teeChan)

	// reduce number of values written to influx
	limiter := server.NewLimiter(5 * time.Second)
	teeChan = limiter.Pipe(teeChan)

	return teeChan
}

// setup influx databases
func configureDatabase(in chan util.Param, conf config) {
	if viper.Get("influx") != nil {
		influx := server.NewInfluxClient(
			conf.Influx.URL,
			conf.Influx.Database,
			conf.Influx.Interval,
			conf.Influx.User,
			conf.Influx.Password,
		)

		teeChan := dbTee(valueChan)
		go influx.Run(teeChan)
	}

	if viper.Get("influx2") != nil {
		influx := server.NewInflux2Client(
			conf.Influx2.URL,
			conf.Influx2.Token,
			conf.Influx2.Org,
			conf.Influx2.Bucket,
		)

		teeChan := dbTee(valueChan)
		go influx.Run(teeChan)
	}
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
