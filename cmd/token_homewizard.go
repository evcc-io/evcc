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
}

func runHomeWizardToken(cmd *cobra.Command, args []string) {
	// Parse log levels to enable debug/trace logging if requested
	parseLogLevels()

	name := cmd.Flag("name").Value.String()
	timeout, _ := cmd.Flags().GetInt("timeout")

	if err := pairing.Run(name, timeout); err != nil {
		log.FATAL.Fatal(err)
	}
}
