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
	conf, err := loadConfigFile(cfgFile)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(conf); err != nil {
		log.FATAL.Fatal(err)
	}

	site, err := configureSiteAndLoadpoints(conf)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	d := dumper{len: 2}

	d.Header("config", "=")
	fmt.Println("")

	if name := site.Meters.GridMeterRef; name != "" {
		d.DumpWithHeader(fmt.Sprintf("grid: %s", name), cp.Meter(name))
	}
	if name := site.Meters.PVMeterRef; name != "" {
		d.DumpWithHeader(fmt.Sprintf("pv: %s", name), cp.Meter(name))
	}
	if name := site.Meters.BatteryMeterRef; name != "" {
		d.DumpWithHeader(fmt.Sprintf("battery: %s", name), cp.Meter(name))
	}

	for id, lpI := range site.LoadPoints() {
		lp := lpI.(*core.LoadPoint)

		d.Header(fmt.Sprintf("loadpoint %d", id+1), "=")
		fmt.Println("")

		if name := lp.Meters.ChargeMeterRef; name != "" {
			d.DumpWithHeader(fmt.Sprintf("charge: %s", name), cp.Meter(name))
		}

		if name := lp.ChargerRef; name != "" {
			d.DumpWithHeader(fmt.Sprintf("charger: %s", name), cp.Charger(name))
		}

		for id, v := range lp.VehiclesRef {
			d.DumpWithHeader(fmt.Sprintf("vehicle %d", id), cp.Vehicle(v))
		}
	}
}
