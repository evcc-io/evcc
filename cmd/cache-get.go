package cmd

import (
	"fmt"
	"regexp"

	"github.com/evcc-io/evcc/server/db/cache"
	"github.com/spf13/cobra"
)

// cacheGetCmd represents the cache get command
var cacheGetCmd = &cobra.Command{
	Use:   "get",
	Short: "Get cache entries",
	Run:   runCacheGet,
	Args:  cobra.MaximumNArgs(1),
}

func init() {
	cacheCmd.AddCommand(cacheGetCmd)
}

func runCacheGet(cmd *cobra.Command, args []string) {
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

	entries, err := cache.All()
	if err != nil {
		log.FATAL.Fatal(err)
	}

	for _, entry := range entries {
		if re != nil && !re.MatchString(entry.Key) {
			continue
		}

		fmt.Printf("%s:\n", entry.Key)
		fmt.Println(entry.Value)
		fmt.Println("---")
	}
}
