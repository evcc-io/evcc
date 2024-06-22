package cmd

import (
	"fmt"
	"os"
	"regexp"
	"text/tabwriter"

	"github.com/evcc-io/evcc/server/db/settings"
	"github.com/spf13/cobra"
)

// settingsGetCmd represents the configure command
var settingsGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get configuration settings",
	Run:   runSettingsGet,
	Args:  cobra.MaximumNArgs(1),
}

func init() {
	settingsCmd.AddCommand(settingsGetCmd)
}

func runSettingsGet(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup persistence
	if err := configureDatabase(conf.Database); err != nil {
		log.FATAL.Fatal(err)
	}

	var re *regexp.Regexp
	if len(args) > 0 {
		re = regexp.MustCompile(args[0])
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 1, ' ', 0)
	for _, s := range settings.All() {
		if re != nil && !re.MatchString(s.Key) {
			continue
		}

		fmt.Fprintf(w, "%s:\t%s\n", s.Key, s.Value)
	}
	w.Flush()
}
