package cmd

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/andig/evcc/core"
	"github.com/andig/evcc/push"
	"github.com/andig/evcc/util"
	"github.com/spf13/viper"
)

func init() {
	rand.Seed(time.Now().UnixNano())
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

func loadConfig(conf config, pushChan chan<- push.Event) *core.Site {
	cp := &ConfigProvider{}
	cp.configure(conf)

	loadPoints := configureLoadPoints(conf, cp, pushChan)
	site := configureSite(conf.Site, cp, loadPoints)

	return site
}

func configureSite(conf map[string]interface{}, cp *ConfigProvider, loadPoints []*core.LoadPoint) *core.Site {
	return core.NewSiteFromConfig(log, cp, conf, loadPoints)
}

func configureLoadPoints(conf config, cp *ConfigProvider, pushChan chan<- push.Event) (loadPoints []*core.LoadPoint) {
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
