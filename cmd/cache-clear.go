package cmd

import (
	"github.com/AlecAivazis/survey/v2"
	"github.com/evcc-io/evcc/server/db/cache"
	"github.com/spf13/cobra"
)

// cacheClearCmd represents the cache clear command
var cacheClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Clear all cache entries",
	Run:   runCacheClear,
}

func init() {
	cacheCmd.AddCommand(cacheClearCmd)
	cacheClearCmd.Flags().BoolP(flagForce, "f", false, "Force (no confirmation)")
}

func runCacheClear(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup persistence
	if err := configureDatabase(conf.Database); err != nil {
		log.FATAL.Fatal(err)
	}

	confirmation, _ := cmd.Flags().GetBool(flagForce)
	if !confirmation {
		prompt := &survey.Confirm{
			Message: "Clear all cache entries",
		}

		if err := survey.AskOne(prompt, &confirmation); err != nil {
			log.FATAL.Fatal(err)
		}
	}

	if confirmation {
		if err := cache.Clear(); err != nil {
			log.FATAL.Fatal(err)
		}
		log.INFO.Println("Cache cleared successfully")
	}

	// wait for shutdown
	<-shutdownDoneC()
}
