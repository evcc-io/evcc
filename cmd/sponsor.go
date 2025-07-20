package cmd

import (
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/spf13/cobra"
)

// sponsorCmd represents the vehicle command
var sponsorCmd = &cobra.Command{
	Use:   "sponsor [name]",
	Short: "Validate sponsor token",
	Args:  cobra.ExactArgs(1),
	Run:   runSponsor,
}

func init() {
	rootCmd.AddCommand(sponsorCmd)
}

func runSponsor(cmd *cobra.Command, args []string) {
	token := args[0]

	if err := sponsor.ConfigureSponsorship(token); err != nil {
		fatal(err)
	}

	log.INFO.Println("sponsorship validated")
}
