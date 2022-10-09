package cmd

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// cmdEcho.AddCommand

// chargerRampCmd represents the charger command
var chargerRampCmd = &cobra.Command{
	Use:   "ramp [name]",
	Short: "Ramp current from 6..16A in configurable steps",
	Run:   runChargerRamp,
}

func init() {
	chargerCmd.AddCommand(chargerRampCmd)

	chargerRampCmd.Flags().StringP(flagDigits, "", "0", "fractional digits (0..2)")
	chargerRampCmd.Flags().StringP(flagDelay, "", "1s", "ramp delay")
}

func ramp(c api.Charger, digits int, delay time.Duration) {
	steps := math.Pow10(digits)
	delta := 1 / steps

	fmt.Printf("delay:\t%s\n", delay)
	fmt.Printf("\n%6s\t%6s\n", "I (A)", "P (W)")

	if err := c.Enable(true); err != nil {
		log.ERROR.Fatalln(err)
	}
	defer func() { _ = c.Enable(false) }()

	for i := 6.0; i <= 16; {
		var err error

		if cc, ok := c.(api.ChargerEx); ok {
			err = cc.MaxCurrentMillis(i)
		} else {
			err = c.MaxCurrent(int64(i))
		}

		time.Sleep(delay)

		var p float64
		if cc, ok := c.(api.Meter); err == nil && ok {
			p, err = cc.CurrentPower()
		}

		if err != nil {
			log.ERROR.Fatalln(err)
		}

		fmt.Printf("%6.3f\t%6.0f\n", i, p)

		i += delta
	}
}

func runChargerRamp(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s", server.FormattedVersion())

	// load config
	if err := loadConfigFile(&conf); err != nil {
		log.FATAL.Fatal(err)
	}

	setLogLevel(cmd)

	// setup environment
	if err := configureEnvironment(cmd, conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// select single charger
	if err := selectByName(cmd, &conf.Chargers); err != nil {
		log.FATAL.Fatal(err)
	}

	if err := cp.configureChargers(conf); err != nil {
		log.FATAL.Fatal(err)
	}

	chargers := cp.chargers
	if len(args) == 1 {
		name := args[0]
		charger, err := cp.Charger(name)
		if err != nil {
			log.FATAL.Fatal(err)
		}
		chargers = map[string]api.Charger{name: charger}
	}

	digits, err := strconv.Atoi(cmd.Flags().Lookup(flagDigits).Value.String())
	if err != nil {
		log.ERROR.Fatalln(err)
	}

	delay, err := time.ParseDuration(cmd.Flags().Lookup(flagDelay).Value.String())
	if err != nil {
		log.ERROR.Fatalln(err)
	}

	for _, c := range chargers {
		if _, ok := c.(api.ChargerEx); digits > 0 && !ok {
			log.ERROR.Fatalln("charger does not support mA control")
		}
		ramp(c, digits, delay)
	}

	// wait for shutdown
	<-shutdownDoneC()
}
