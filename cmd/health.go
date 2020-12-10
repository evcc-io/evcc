// +build !windows

package cmd

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/tv42/httpunix"
)

const serviceName = "evcc"

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

	u := &httpunix.Transport{
		DialTimeout:           100 * time.Millisecond,
		RequestTimeout:        1 * time.Second,
		ResponseHeaderTimeout: 1 * time.Second,
	}

	u.RegisterLocation(serviceName, server.SocketPath)

	var client = http.Client{
		Transport: u,
	}

	var ok bool
	resp, err := client.Get(fmt.Sprintf("http+unix://%s/health", serviceName))

	if err == nil {
		resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			log.INFO.Printf("health check ok")
			ok = true
		}
	}

	if !ok {
		log.ERROR.Printf("health check failed")
		os.Exit(1)
	}
}
