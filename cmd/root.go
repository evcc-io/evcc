package cmd

import (
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof" // pprof handler
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/server/modbus"
	"github.com/evcc-io/evcc/server/updater"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/pipe"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/evcc-io/evcc/util/telemetry"
	"github.com/fatih/structs"
	"github.com/jeremywohl/flatten"
	"golang.org/x/exp/maps"

	_ "github.com/joho/godotenv/autoload"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const rebootDelay = 5 * time.Minute // delayed reboot on error

var (
	log     = util.NewLogger("main")
	cfgFile string

	ignoreErrors = []string{"warn", "error"}        // don't add to cache
	ignoreMqtt   = []string{"auth", "releaseNotes"} // excessive size may crash certain brokers
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "evcc",
	Short:   "EV Charge Controller",
	Version: server.FormattedVersion(),
	Run:     runRoot,
}

func init() {
	cobra.OnInitialize(initConfig)

	// global options
	rootCmd.PersistentFlags().StringVarP(&cfgFile, "config", "c", "", "Config file (default \"~/evcc.yaml\" or \"/etc/evcc.yaml\")")

	rootCmd.PersistentFlags().BoolP("help", "h", false, "Help")

	rootCmd.PersistentFlags().Bool(flagHeaders, false, flagHeadersDescription)

	// config file options
	rootCmd.PersistentFlags().StringP("log", "l", "info", "Log level (fatal, error, warn, info, debug, trace)")
	bindP(rootCmd, "log")

	rootCmd.Flags().Bool("metrics", false, "Expose metrics")
	bind(rootCmd, "metrics")

	rootCmd.Flags().Bool("profile", false, "Expose pprof profiles")
	bind(rootCmd, "profile")
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

	viper.SetEnvPrefix("evcc")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv() // read in environment variables that match

	// register all known config keys
	flat, _ := flatten.Flatten(structs.Map(conf), "", flatten.DotStyle)
	for _, v := range maps.Keys(flat) {
		_ = viper.BindEnv(v)
	}

	// print version
	util.LogLevel("info", nil)
	log.INFO.Printf("evcc %s", server.FormattedVersion())
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func runRoot(cmd *cobra.Command, args []string) {
	// load config and re-configure logging after reading config file
	var err error
	if cfgErr := loadConfigFile(&conf); errors.As(cfgErr, &viper.ConfigFileNotFoundError{}) {
		log.INFO.Println("missing config file - switching into demo mode")
		if err := demoConfig(&conf); err != nil {
			log.FATAL.Fatal(err)
		}
	} else {
		err = cfgErr
	}

	// network config
	if viper.GetString("uri") != "" {
		log.WARN.Println("`uri` is deprecated and will be ignored. Use `network` instead.")
	}

	log.INFO.Printf("starting ui and api at :%d", conf.Network.Port)

	// start broadcasting values
	tee := new(util.Tee)

	// value cache
	cache := util.NewCache()
	go cache.Run(pipe.NewDropper(ignoreErrors...).Pipe(tee.Attach()))

	// create web server
	socketHub := server.NewSocketHub()
	httpd := server.NewHTTPd(fmt.Sprintf(":%d", conf.Network.Port), socketHub)

	// metrics
	if viper.GetBool("metrics") {
		httpd.Router().Handle("/metrics", promhttp.Handler())
	}

	// pprof
	if viper.GetBool("profile") {
		httpd.Router().PathPrefix("/debug/").Handler(http.DefaultServeMux)
	}

	// publish to UI
	go socketHub.Run(tee.Attach(), cache)

	// setup values channel
	valueChan := make(chan util.Param)
	go tee.Run(valueChan)

	// capture log messages for UI
	util.CaptureLogs(valueChan)

	// setup environment
	if err == nil {
		err = configureEnvironment(cmd, conf)
	}

	// setup telemetry
	if err == nil {
		telemetry.Create(conf.Plant)
		if conf.Telemetry {
			err = telemetry.Enable(true)
		}
	}

	// setup modbus proxy
	if err == nil {
		for _, cfg := range conf.ModbusProxy {
			if err = modbus.StartProxy(cfg.Port, cfg.Settings, cfg.ReadOnly); err != nil {
				break
			}
		}
	}

	// setup site and loadpoints
	var site *core.Site
	if err == nil {
		cp.TrackVisitors() // track duplicate usage
		site, err = configureSiteAndLoadpoints(conf)
	}

	// setup database
	if err == nil && conf.Influx.URL != "" {
		configureInflux(conf.Influx, site, tee.Attach())
	}

	// setup mqtt publisher
	if err == nil && conf.Mqtt.Broker != "" {
		publisher := server.NewMQTT(strings.Trim(conf.Mqtt.Topic, "/"))
		go publisher.Run(site, pipe.NewDropper(ignoreMqtt...).Pipe(tee.Attach()))
	}

	// announce on mDNS
	if err == nil && strings.HasSuffix(conf.Network.Host, ".local") {
		err = configureMDNS(conf.Network)
	}

	// start HEMS server
	if err == nil && conf.HEMS.Type != "" {
		err = configureHEMS(conf.HEMS, site, httpd)
	}

	// setup messaging
	var pushChan chan push.Event
	if err == nil {
		pushChan, err = configureMessengers(conf.Messaging, valueChan, cache)
	}

	// run shutdown functions on stop
	var once sync.Once
	stopC := make(chan struct{})

	// catch signals
	go func() {
		signalC := make(chan os.Signal, 1)
		signal.Notify(signalC, os.Interrupt, syscall.SIGTERM)

		<-signalC                        // wait for signal
		once.Do(func() { close(stopC) }) // signal loop to end
	}()

	// wait for shutdown
	go func() {
		<-stopC

		select {
		case <-shutdownDoneC(): // wait for shutdown
		case <-time.After(conf.Interval):
		}

		if err != nil {
			os.Exit(1)
		}

		os.Exit(0)
	}()

	// show main ui
	if err == nil {
		httpd.RegisterSiteHandlers(site, cache)

		// set channels
		site.DumpConfig()
		site.Prepare(valueChan, pushChan)

		// show and check version
		valueChan <- util.Param{Key: "version", Val: server.FormattedVersion()}
		go updater.Run(log, httpd, valueChan)

		// expose sponsor to UI
		if sponsor.Subject != "" {
			valueChan <- util.Param{Key: "sponsor", Val: sponsor.Subject}
		}

		// allow web access for vehicles
		cp.webControl(conf.Network, httpd.Router(), valueChan)

		go func() {
			site.Run(stopC, conf.Interval)
		}()
	} else {
		httpd.RegisterShutdownHandler(func() {
			log.FATAL.Println("evcc was stopped. OS should restart the service. Or restart manually.")
			once.Do(func() { close(stopC) }) // signal loop to end
		})

		// improve error message
		err = wrapErrors(err)

		publishErrorInfo(valueChan, cfgFile, err)

		log.FATAL.Println(err)
		log.FATAL.Printf("will attempt restart in: %v", rebootDelay)

		go func() {
			<-time.After(rebootDelay)
			once.Do(func() { close(stopC) }) // signal loop to end
		}()
	}

	// uds health check listener
	go server.HealthListener(site)

	log.FATAL.Println(wrapErrors(httpd.ListenAndServe()))
}
