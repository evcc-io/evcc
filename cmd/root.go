package cmd

import (
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
		// Search for config in home directory if available
		if home, err := os.UserHomeDir(); err == nil {
			viper.AddConfigPath(home)
		}

		// Search config in home directory with name "mbmd" (without extension).
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
	api.OutThreshold = api.LogLevelToThreshold(level)
	api.LogThreshold = api.OutThreshold
	api.Loggers(func(name string, logger *api.Logger) {
		logger.SetStdoutThreshold(api.OutThreshold)
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

var teeIsChained bool // controles piping of first channel in teed chain

// tee splits a tee channel from an input channel and starts a goroutine
// for duplicateing channel values to replacement input and tee channel
func tee(in chan core.Param) (chan core.Param, <-chan core.Param) {
	gen := make(chan core.Param)
	tee := make(chan core.Param)

	go func(teeIsChained bool) {
		for i := range gen {
			if teeIsChained {
				in <- i
			}
			tee <- i
		}
	}(teeIsChained)

	teeIsChained = true
	return gen, tee
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

	// setup messaging
	notificationChan := configureMessengers(conf.Messaging)

	// setup loadpoints
	loadPoints := loadConfig(conf, notificationChan)

	// start broadcasting values
	valueChan := make(chan core.Param)
	triggerChan := make(chan struct{})

	// setup influx
	if viper.Get("influx") != nil {
		influx := server.NewInfluxClient(
			conf.Influx.URL,
			conf.Influx.Database,
			conf.Influx.Interval,
			conf.Influx.User,
			conf.Influx.Password,
		)

		var teeChan <-chan core.Param
		valueChan, teeChan = tee(valueChan)

		// eliminate duplicate values
		dedupe := server.NewDeduplicator(30*time.Minute, "socCharge")
		teeChan = dedupe.Pipe(teeChan)

		// reduce number of values written to influx
		limiter := server.NewLimiter(5 * time.Second)
		teeChan = limiter.Pipe(teeChan)

		go influx.Run(teeChan)
	}

	// create webserver
	socketHub := server.NewSocketHub()
	httpd := server.NewHttpd(uri, conf.Menu, loadPoints[0], socketHub)

	var teeChan <-chan core.Param
	valueChan, teeChan = tee(valueChan)

	go socketHub.Run(teeChan, triggerChan)

	// start all loadpoints
	for _, lp := range loadPoints {
		lp.Dump()
		lp.Prepare(valueChan, notificationChan)
		go lp.Run(conf.Interval)
	}

	// handle UI update requests whenever browser connects
	go func() {
		for range triggerChan {
			for _, lp := range loadPoints {
				lp.Update()
			}
		}
	}()

	log.FATAL.Println(httpd.ListenAndServe())
}
