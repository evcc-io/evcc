package cmd

import (
	"net/http"
	"os"
	"strings"

	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// healthCmd represents the meter command
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "Check application health",
	Run:   runHealth,
}

func init() {
	rootCmd.AddCommand(healthCmd)
}

func runHealth(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config
	conf := loadConfigFile(cfgFile)

	uri := strings.TrimRight(conf.URI, "/") + "/api/health"
	if !strings.HasPrefix(uri, "http") {
		uri = "http://" + uri
	}

	var ok bool
	resp, err := http.Get(uri)
	if err == nil && resp.StatusCode == http.StatusOK {
		log.INFO.Printf("health check ok")
		ok = true
	}

	if !ok {
		log.ERROR.Printf("health check failed")
		os.Exit(1)
	}
}
