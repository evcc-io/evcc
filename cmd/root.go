package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/core"
	"github.com/andig/evcc/server"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	latest "github.com/tcnksm/go-latest"
)

var (
	log     = api.NewLogger("main")
	cfgFile string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "evcc",
	Short: "EV Charge Controller",
	Run:   run,
}

func bind(flag string) {
	if err := viper.BindPFlag(flag, rootCmd.PersistentFlags().Lookup(flag)); err != nil {
		panic(err)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.PersistentFlags().StringP(
		"uri", "u",
		"0.0.0.0:7070",
		"Listen address",
	)
	bind("uri")

	rootCmd.PersistentFlags().StringP(
		"log", "l",
		"info",
		"Log level (fatal, error, warn, info, debug, trace)",
	)
	bind("log")

	rootCmd.PersistentFlags().DurationP(
		"interval", "i",
		10*time.Second,
		"Update interval",
	)
	bind("interval")

	rootCmd.PersistentFlags().StringVarP(&cfgFile,
		"config", "c",
		"",
		"Config file (default \"~/evcc.yaml\" or \"/etc/evcc.yaml\")",
	)
	rootCmd.PersistentFlags().BoolP(
		"help", "h",
		false,
		"Help for "+rootCmd.Name(),
	)
}

// initConfig reads in config file and ENV variables if set
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory
		home, err := os.UserHomeDir()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		// Search config in home directory with name "mbmd" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")    // optionally look for config in the working directory
		viper.AddConfigPath("/etc") // path to look for the config file in

		viper.SetConfigName("evcc")
	}

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in
	if err := viper.ReadInConfig(); err == nil {
		// using config file
		cfgFile = viper.ConfigFileUsed()
	} else if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
		// parsing failed - exit
		fmt.Println(err)
		os.Exit(1)
	} else {
		// not using config file
		cfgFile = ""
	}
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func configureLogging(level string) {
	// f, err := os.OpenFile("evcc.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	// if err != nil {
	// 	log.FATAL.Fatal(err)
	// }

	api.OutThreshold = api.LogLevelToThreshold(level)
	api.LogThreshold = api.OutThreshold
	api.Loggers(func(name string, logger *api.Logger) {
		logger.SetStdoutThreshold(api.OutThreshold)
		// logger.SetLogThreshold(api.OutThreshold)
		// logger.SetLogOutput(f)
	})
}

// checkVersion validates if updates are available
func checkVersion() {
	githubTag := &latest.GithubTag{
		Owner:      "andig",
		Repository: "evcc",
	}

	if res, err := latest.Check(githubTag, server.Version); err == nil {
		if res.Outdated {
			log.INFO.Printf("updates available - please upgrade to %s", res.Current)
		}
	}
}

// cycle executes loadpoint update and publishes all paramters
func cycle(lp *core.LoadPoint, tick time.Duration, updateChan chan core.Param) {
	lp.Update()

	ctx, cancel := context.WithTimeout(context.Background(), tick)
	lp.Publish(ctx, updateChan)
	cancel()
}

func run(cmd *cobra.Command, args []string) {
	level, _ := cmd.PersistentFlags().GetString("log")
	configureLogging(level)
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	var conf config
	if cfgFile != "" {
		log.INFO.Println("using config file", cfgFile)
		if err := viper.UnmarshalExact(&conf); err != nil {
			log.FATAL.Fatalf("config: failed parsing config file %s: %v", cfgFile, err)
		}
	} else {
		log.FATAL.Fatal("missing evcc config")
	}

	// re-configure after reading config file
	configureLogging(conf.Log)

	go checkVersion()

	uri := viper.GetString("uri")
	log.INFO.Println("listening at", uri)
	loadPoints := loadConfig(conf)

	// setup messaging
	// if conf.Pushover.App != "" {
	// 	po := server.NewMessenger(conf.Pushover.App, conf.Pushover.Recipients)
	// 	po.Send("Wallbe", "Charge started", "Charging started in PV mode")
	// }

	// create webserver
	hub := server.NewSocketHub()
	httpd := server.NewHttpd(uri, conf.Menu, loadPoints[0], hub)

	// start broadcasting values
	updateChan := make(chan core.Param)
	go hub.Run(updateChan)

	for _, lp := range loadPoints {
		lp.Dump()

		// update loop
		go func(lp *core.LoadPoint) {
			tick := conf.Interval
			cycle(lp, tick, updateChan)

			for range time.Tick(tick) {
				cycle(lp, tick, updateChan)
			}
		}(lp)
	}

	log.FATAL.Println(httpd.ListenAndServe())
}
