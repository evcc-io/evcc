package cmd

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/util/config"
	"github.com/spf13/cobra"
	"golang.org/x/exp/slices"
)

// deviceCmd represents the device debug command
var deviceCmd = &cobra.Command{
	Use:   "device",
	Short: "Query database-configured devices (debug only)",
	Run:   runDevice,
}

func init() {
	rootCmd.AddCommand(deviceCmd)
}

func runDevice(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, conf); err != nil {
		log.FATAL.Fatal(err)
	}

	for _, class := range []config.Class{config.Meter, config.Charger, config.Vehicle} {
		devs, err := config.Devices(class)
		if err != nil {
			log.FATAL.Fatal(err)
		}

		if len(devs) == 0 {
			continue
		}

		fmt.Println(class)
		fmt.Println(strings.Repeat("-", len(class.String())))

		for _, d := range devs {
			fmt.Printf("%d. %s\n", d.ID, d.Type)

			details := d.Details
			slices.SortFunc(details, func(i, j config.DeviceDetail) bool {
				return i.Key < j.Key
			})

			for _, d := range details {
				fmt.Printf("%s: %s\n", d.Key, d.Value)
			}

			fmt.Println()
		}
	}

	// wait for shutdown
	<-shutdownDoneC()
}
