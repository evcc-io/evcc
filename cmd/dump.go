package cmd

import (
	"fmt"

	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
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
	log.INFO.Printf("evcc %s", server.FormattedVersion())

	// load config
	if err := loadConfigFile(cfgFile, &conf); err != nil {
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

	if len(site.Meters.PVMetersRef) == 0 {
		if name := site.Meters.PVMeterRef; name != "" {
			d.DumpWithHeader(fmt.Sprintf("pv: %s", name), cp.Meter(name))
		}
	} else {
		for id, name := range site.Meters.PVMetersRef {
			if name != "" {
				d.DumpWithHeader(fmt.Sprintf("pv %d: %s", id, name), cp.Meter(name))
			}
		}
	}

	if len(site.Meters.BatteryMetersRef) == 0 {
		if name := site.Meters.BatteryMeterRef; name != "" {
			d.DumpWithHeader(fmt.Sprintf("battery: %s", name), cp.Meter(name))
		}
	} else {
		for id, name := range site.Meters.BatteryMetersRef {
			if name != "" {
				d.DumpWithHeader(fmt.Sprintf("battery %d: %s", id, name), cp.Meter(name))
			}
		}
	}

	for _, v := range site.GetVehicles() {
		d.DumpWithHeader(fmt.Sprintf("vehicle: %s", v.Title()), v)
	}

	for id, lpI := range site.LoadPoints() {
		lp := lpI.(*core.LoadPoint)

		d.Header(fmt.Sprintf("loadpoint %d", id+1), "=")
		fmt.Println("")

		if name := lp.MeterRef; name != "" {
			d.DumpWithHeader(fmt.Sprintf("charge: %s", name), cp.Meter(name))
		}

		if name := lp.ChargerRef; name != "" {
			d.DumpWithHeader(fmt.Sprintf("charger: %s", name), cp.Charger(name))
		}
	}
}
