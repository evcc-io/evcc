package cmd

import (
	"fmt"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// twcCmd represents the meter command
var twcCmd = &cobra.Command{
	Use:   "twc",
	Short: "TWC2 test",
	Run:   runMeterTWC,
}

func init() {
	rootCmd.AddCommand(twcCmd)
}

func runMeterTWC(cmd *cobra.Command, args []string) {
	util.LogLevel("trace")
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config
	conf := loadConfigFile(cfgFile)

	// setup mqtt
	if viper.Get("mqtt") != nil {
		provider.MQTT = provider.NewMqttClient(conf.Mqtt.Broker, conf.Mqtt.User, conf.Mqtt.Password, clientID(), 1)
	}

	cp := &ConfigProvider{}
	cp.configureChargers(conf)

	chargers := cp.chargers
	if len(args) == 1 {
		arg := args[0]
		chargers = map[string]api.Charger{arg: cp.Charger(arg)}
	}

	for name, v := range chargers {
		if len(chargers) != 1 {
			fmt.Println(name)
		}

		if status, err := v.Status(); err != nil {
			fmt.Printf("Status: %v\n", err)
		} else {
			fmt.Printf("Status: %s\n", status)
		}

		if enabled, err := v.Enabled(); err != nil {
			fmt.Printf("Enabled: %v\n", err)
		} else {
			fmt.Printf("Enabled: %s\n", truefalse[enabled])
		}

		dumpAPIs(v)
	}

	time.Sleep(10 * time.Second)
}
