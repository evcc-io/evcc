// +build !windows

package cmd

import (
	"fmt"
	"net/http"
	"os"
	"strings"
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

func udsRequest(uri string) (*http.Response, error) {
	u := &httpunix.Transport{
		DialTimeout:           100 * time.Millisecond,
		RequestTimeout:        1 * time.Second,
		ResponseHeaderTimeout: 1 * time.Second,
	}

	u.RegisterLocation(serviceName, server.SocketPath)

	var client = http.Client{
		Transport: u,
	}

	return client.Get(fmt.Sprintf("http+unix://%s/%s", serviceName, strings.TrimLeft(uri, "/")))
}

func runHealth(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config
	// conf := loadConfigFile(cfgFile)

	// uri := strings.TrimRight(conf.URI, "/") + "/api/health"
	// if !strings.HasPrefix(uri, "http") {
	// 	uri = "http://" + uri
	// }
	// resp, err := http.Get(uri)

	var ok bool
	resp, err := udsRequest("/health")

	if err == nil && resp.StatusCode == http.StatusOK {
		log.INFO.Printf("health check ok")
		ok = true
	}

	if !ok {
		log.ERROR.Printf("health check failed")
		os.Exit(1)
	}
}
