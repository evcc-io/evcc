package cmd

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andig/evcc/server"
	"github.com/andig/evcc/server/updater"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/pipe"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	ignoreParams = []string{"warn", "error", "fatal"} // don't add to cache
	log          = util.NewLogger("main")
	cfgFile      string
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "evcc",
	Short:   "EV Charge Controller",
	Version: fmt.Sprintf("%s (%s)", server.Version, server.Commit),
	Run:     run,
}

func bind(cmd *cobra.Command, flag string) {
	if err := viper.BindPFlag(flag, cmd.PersistentFlags().Lookup(flag)); err != nil {
		panic(err)
	}
}

func configureCommand(cmd *cobra.Command) {
	cmd.PersistentFlags().StringP(
		"log", "l",
		"error",
		"Log level (fatal, error, warn, info, debug, trace)",
	)
	bind(cmd, "log")

	cmd.PersistentFlags().StringVarP(&cfgFile,
		"config", "c",
		"",
		"Config file (default \"~/evcc.yaml\" or \"/etc/evcc.yaml\")",
	)
	cmd.PersistentFlags().BoolP(
		"help", "h",
		false,
		"Help for "+cmd.Name(),
	)
}

func init() {
	cobra.OnInitialize(initConfig)
	configureCommand(rootCmd)

	rootCmd.PersistentFlags().StringP(
		"uri", "u",
		"0.0.0.0:7070",
		"Listen address",
	)
	bind(rootCmd, "uri")

	rootCmd.PersistentFlags().DurationP(
		"interval", "i",
		10*time.Second,
		"Update interval",
	)
	bind(rootCmd, "interval")
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

func run(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config and re-configure logging after reading config file
	conf := loadConfigFile(cfgFile)
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))

	uri := viper.GetString("uri")
	log.INFO.Println("listening at", uri)

	// setup mqtt client listener
	if conf.Mqtt.Broker != "" {
		configureMQTT(conf.Mqtt)
	}

	// start broadcasting values
	tee := &Tee{}

	// value cache
	cache := util.NewCache()
	go cache.Run(pipe.NewDropper(ignoreParams...).Pipe(tee.Attach()))

	// setup loadpoints
	site, err := loadConfig(conf)
	if err != nil {
		cp.Close()
		log.FATAL.Fatal(err)
	}

	// setup database
	if conf.Influx.URL != "" {
		configureDatabase(conf.Influx, site.LoadPoints(), tee.Attach())
	}

	// setup mqtt publisher
	if conf.Mqtt.Broker != "" {
		publisher := server.NewMQTT(conf.Mqtt.Topic)
		go publisher.Run(site, tee.Attach())
	}

	// create webserver
	socketHub := server.NewSocketHub()
	httpd := server.NewHTTPd(uri, site, socketHub, cache)

	// start HEMS server
	if conf.HEMS != "" {
		hems := configureHEMS(conf.HEMS, site, cache, httpd)
		go hems.Run()
	}

	// publish to UI
	go socketHub.Run(tee.Attach(), cache)

	// setup values channel
	valueChan := make(chan util.Param)
	go tee.Run(valueChan)

	// version check
	go updater.Run(log, valueChan)

	// capture log messages for UI
	util.CaptureLogs(valueChan)

	// setup messaging
	pushChan := configureMessengers(conf.Messaging, cache)

	// set channels
	site.Prepare(valueChan, pushChan)
	site.DumpConfig()

	stopC := make(chan struct{})
	exitC := make(chan struct{})

	go func() {
		site.Run(stopC, conf.Interval)
		close(exitC)
	}()

	// uds health check listener
	go server.HealthListener(site)

	// catch signals
	go func() {
		signalC := make(chan os.Signal, 1)
		signal.Notify(signalC, os.Interrupt, syscall.SIGTERM)

		<-signalC    // wait for signal
		close(stopC) // signal loop to end
		<-exitC      // wait for loop to end
		cp.Close()   // cleanup

		os.Exit(1)
	}()

	log.FATAL.Println(httpd.ListenAndServe())
}
