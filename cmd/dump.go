package cmd

import (
	"fmt"

	"github.com/andig/evcc/core"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// dumpCmd represents the meter command
var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Dump configuration",
	Run:   runDump,
}

func init() {
	rootCmd.AddCommand(dumpCmd)
}

func runDump(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config
	conf := loadConfigFile(cfgFile)

	// setup mqtt
	if conf.Mqtt.Broker != "" {
		configureMQTT(conf.Mqtt)
	}

	site, err := loadConfig(conf)
	if err != nil {
		cp.Close() // cleanup any open sessions
		log.FATAL.Fatal(err)
	}

	defer cp.Close() // cleanup on exit

	d := dumper{len: 2}

	d.Header("config", "=")
	fmt.Println("")

	if site.Meters.GridMeterRef != "" {
		d.DumpWithHeader("grid", cp.Meter(site.Meters.GridMeterRef))
	}
	if site.Meters.PVMeterRef != "" {
		d.DumpWithHeader("pv", cp.Meter(site.Meters.PVMeterRef))
	}
	if site.Meters.BatteryMeterRef != "" {
		d.DumpWithHeader("battery", cp.Meter(site.Meters.BatteryMeterRef))
	}

	for id, lpI := range site.LoadPoints() {
		lp := lpI.(*core.LoadPoint)

		d.Header(fmt.Sprintf("loadpoint %d", id+1), "=")
		fmt.Println("")

		if lp.Meters.ChargeMeterRef != "" {
			d.DumpWithHeader("charge", cp.Meter(lp.Meters.ChargeMeterRef))
		}

		if lp.ChargerRef != "" {
			d.DumpWithHeader("charger", cp.Charger(lp.ChargerRef))
		}

		for id, v := range lp.VehiclesRef {
			d.DumpWithHeader(fmt.Sprintf("vehicle %d", id), cp.Vehicle(v))
		}
	}
}
