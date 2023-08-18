package cmd

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cobra"
)

// chargerRampCmd represents the charger command
var chargerRampCmd = &cobra.Command{
	Use:   "ramp [name]",
	Short: "Ramp current from 6..16A in configurable steps",
	Args:  cobra.MaximumNArgs(1),
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
	// load config
	if err := loadConfigFile(&conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// select single charger
	if err := selectByName(args, &conf.Chargers); err != nil {
		log.FATAL.Fatal(err)
	}

	if err := configureChargers(conf.Chargers); err != nil {
		log.FATAL.Fatal(err)
	}

	digits, err := strconv.Atoi(cmd.Flags().Lookup(flagDigits).Value.String())
	if err != nil {
		log.ERROR.Fatalln(err)
	}

	delay, err := time.ParseDuration(cmd.Flags().Lookup(flagDelay).Value.String())
	if err != nil {
		log.ERROR.Fatalln(err)
	}

	chargers := config.Chargers().Devices()

	for _, v := range config.Instances(chargers) {
		if _, ok := v.(api.ChargerEx); digits > 0 && !ok {
			log.ERROR.Fatalln("charger does not support mA control")
		}
		ramp(v, digits, delay)
	}

	// wait for shutdown
	<-shutdownDoneC()
}
