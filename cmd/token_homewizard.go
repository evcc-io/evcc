package cmd

import (
	"github.com/mluiten/evcc-homewizard-v2/pairing"
	"github.com/spf13/cobra"
)

var homeWizardCmd = &cobra.Command{
	Use:   "homewizard",
	Short: "Pair with HomeWizard devices using v2 API",
	Run:   runHomeWizardToken,
}

func init() {
	tokenCmd.AddCommand(homeWizardCmd)
	homeWizardCmd.Flags().StringP("name", "n", "evcc", "Product name for pairing")
	homeWizardCmd.Flags().Int("timeout", 10, "Discovery timeout in seconds")
	homeWizardCmd.Flags().String("host", "", "Device host/IP (skips discovery, pairs specific device)")
}

func runHomeWizardToken(cmd *cobra.Command, args []string) {
	// Parse log levels to enable debug/trace logging if requested
	parseLogLevels()

	name := cmd.Flag("name").Value.String()
	timeout, _ := cmd.Flags().GetInt("timeout")
	host, _ := cmd.Flags().GetString("host")

	// If host is specified, skip discovery and pair directly
	if host != "" {
		if err := pairing.PairSingleDevice(host, name); err != nil {
			log.FATAL.Fatal(err)
		}
		return
	}

	// Otherwise run full discovery + pairing flow
	if err := pairing.DiscoverAndPairDevices(name, timeout); err != nil {
		log.FATAL.Fatal(err)
	}
}
