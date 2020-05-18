package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/andig/evcc/core"
	"github.com/andig/evcc/provider"
	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	latest "github.com/tcnksm/go-latest"
)

var (
	log     = util.NewLogger("main")
	cfgFile string
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

// handle UI update requests
func handleUI(triggerChan <-chan struct{}, loadPoints []*core.LoadPoint) {
	for range triggerChan {
		for _, lp := range loadPoints {
			lp.Update()
		}
	}
}

func run(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config and re-configure logging after reading config file
	conf := loadConfigFile(cfgFile)
	util.LogLevel(viper.GetString("log"))

	go checkVersion()

	uri := viper.GetString("uri")
	log.INFO.Println("listening at", uri)

	// setup messaging
	notificationChan := configureMessengers(conf.Messaging)

	// setup mqtt
	if viper.Get("mqtt") != nil {
		provider.MQTT = provider.NewMqttClient(conf.Mqtt.Broker, conf.Mqtt.User, conf.Mqtt.Password, clientID(), 1)
	}

	// setup loadpoints
	loadPoints := loadConfig(conf, notificationChan)

	// start broadcasting values
	tee := &Tee{}

	// setup influx
	if viper.Get("influx") != nil {
		influx := server.NewInfluxClient(
			conf.Influx.URL,
			conf.Influx.Database,
			conf.Influx.Interval,
			conf.Influx.User,
			conf.Influx.Password,
		)

		// eliminate duplicate values
		dedupe := server.NewDeduplicator(30*time.Minute, "socCharge")
		pipeChan := dedupe.Pipe(tee.Attach())

		// reduce number of values written to influx
		limiter := server.NewLimiter(5 * time.Second)
		pipeChan = limiter.Pipe(pipeChan)

		go influx.Run(pipeChan)
	}

	// create webserver
	socketHub := server.NewSocketHub()
	httpd := server.NewHTTPd(uri, conf.Menu, loadPoints[0], socketHub)

	triggerChan := make(chan struct{})

	// handle UI update requests whenever browser connects
	go handleUI(triggerChan, loadPoints)

	// publish to UI
	go socketHub.Run(tee.Attach(), triggerChan)

	// setup values channel
	valueChan := make(chan util.Param)
	go tee.Run(valueChan)

	// capture log messages for UI
	util.CaptureLogs(valueChan)

	// start all loadpoints
	for _, lp := range loadPoints {
		lp.Dump()
		lp.Prepare(valueChan, notificationChan)
		go lp.Run(conf.Interval)
	}

	log.FATAL.Println(httpd.ListenAndServe())
}
