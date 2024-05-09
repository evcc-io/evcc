package cmd

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cobra"
)

// meterCmd represents the meter command
var meterCmd = &cobra.Command{
	Use:   "meter [name]",
	Short: "Query configured meters",
	Args:  cobra.MaximumNArgs(1),
	Run:   runMeter,
}

func init() {
	rootCmd.AddCommand(meterCmd)
	meterCmd.Flags().StringP(flagBatteryMode, "b", "", flagBatteryModeDescription)
	meterCmd.Flags().DurationP(flagBatteryModeWait, "w", 0, flagBatteryModeWaitDescription)
	meterCmd.Flags().BoolP(flagRepeat, "r", false, flagRepeatDescription)
}

func runMeter(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, conf); err != nil {
		log.FATAL.Fatal(err)
	}

	if err := configureMeters(conf.Meters, args...); err != nil {
		log.FATAL.Fatal(err)
	}

	mode := api.BatteryUnknown
	if val := cmd.Flags().Lookup(flagBatteryMode).Value.String(); val != "" {
		var err error
		mode, err = api.BatteryModeString(val)
		if err != nil {
			log.ERROR.Fatalln(err)
		}
	}

	meters := config.Meters().Devices()

	var flagUsed bool
	if mode != api.BatteryUnknown {
		flagUsed = true

		for _, v := range config.Instances(meters) {
			if b, ok := v.(api.BatteryController); ok {
				if err := b.SetBatteryMode(mode); err != nil {
					log.FATAL.Fatalln("set battery mode:", err)
				}
			}

			if d, err := cmd.Flags().GetDuration(flagBatteryModeWait); d > 0 && err == nil {
				log.INFO.Println("waiting for:", d)
				time.Sleep(d)
			}
		}
	}

	if !flagUsed {
		d := dumper{len: len(meters)}
	REPEAT:
		for _, dev := range meters {
			v := dev.Instance()

			d.DumpWithHeader(dev.Config().Name, v)
		}
		if ok, _ := cmd.Flags().GetBool(flagRepeat); ok {
			goto REPEAT
		}
	}

	// wait for shutdown
	<-shutdownDoneC()
}
