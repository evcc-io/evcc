package cmd

import (
	"errors"
	"fmt"
	"net/http"
	_ "net/http/pprof" // pprof handler
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/evcc-io/evcc/cmd/shutdown"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/push"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/server/updater"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/pipe"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	log     = util.NewLogger("main")
	cfgFile string

	ignoreErrors = []string{"warn", "error"}        // don't add to cache
	ignoreMqtt   = []string{"auth", "releaseNotes"} // excessive size may crash certain brokers
)

var conf = config{
	Network: networkConfig{
		Schema: "http",
		Host:   "evcc.local",
		Port:   7070,
	},
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:     "evcc",
	Short:   "EV Charge Controller",
	Version: server.FormattedVersion(),
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
		"info",
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

	rootCmd.PersistentFlags().IntP("port", "p", 7070, "Listen port")
	if err := viper.BindPFlag("network.port", rootCmd.PersistentFlags().Lookup("port")); err != nil {
		panic(err)
	}

	rootCmd.PersistentFlags().DurationP("interval", "i", 10*time.Second, "Update interval")
	bind(rootCmd, "interval")

	rootCmd.PersistentFlags().Bool("metrics", false, "Expose metrics")
	bind(rootCmd, "metrics")

	rootCmd.PersistentFlags().Bool("profile", false, "Expose pprof profiles")
	bind(rootCmd, "profile")

	rootCmd.PersistentFlags().Bool(flagHeaders, false, flagHeadersDescription)
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
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

var valueChan chan util.Param

func publish(key string, val any) {
	valueChan <- util.Param{Key: key, Val: val}
}

func unwrap(err error) (res []string) {
	for err != nil {
		inner := errors.Unwrap(err)
		if inner == nil {
			res = append(res, err.Error())
		} else {
			cur := strings.TrimSuffix(err.Error(), ": "+inner.Error())
			cur = strings.TrimSuffix(cur, inner.Error())
			res = append(res, strings.TrimSpace(cur))
		}
		err = inner
	}
	return
}

func redact(src string) string {
	secrets := []string{
		"url", "uri", "host", "broker", // infrastructure
		"user", "password", // users
		"token", "access", "refresh", "sponsortoken", // tokens
		"ain", "id", "secret", "serial", "deviceid", "machineid", // devices
		"vin"} // vehicles
	return regexp.
		MustCompile(fmt.Sprintf(`\b(%s)\b.*?:.*`, strings.Join(secrets, "|"))).
		ReplaceAllString(src, "$1: *****")
}

func publishErrorInfo(cfgFile string, err error) {
	if cfgFile != "" {
		file, pathErr := filepath.Abs(cfgFile)
		if pathErr != nil {
			file = cfgFile
		}
		publish("file", file)

		if src, fileErr := os.ReadFile(cfgFile); fileErr != nil {
			log.ERROR.Println("could not open config file:", fileErr)
		} else {
			publish("config", redact(string(src)))

			// find line number
			if match := regexp.MustCompile(`yaml: line (\d+):`).FindStringSubmatch(err.Error()); len(match) == 2 {
				if line, err := strconv.Atoi(match[1]); err == nil {
					publish("line", line)
				}
			}
		}
	}

	publish("fatal", unwrap(err))
}

func run(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s", server.FormattedVersion())

	// load config and re-configure logging after reading config file
	var err error
	if cfgErr := loadConfigFile(&conf); errors.As(cfgErr, &viper.ConfigFileNotFoundError{}) {
		log.INFO.Println("missing config file - switching into demo mode")
		demoConfig(&conf)
	} else {
		err = cfgErr
	}

	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))

	// full http request log
	if cmd.PersistentFlags().Lookup(flagHeaders).Changed {
		request.LogHeaders = true
	}

	// network config
	if viper.GetString("uri") != "" {
		log.WARN.Println("`uri` is deprecated and will be ignored. Use `network` instead.")
	}

	if cmd.PersistentFlags().Lookup("port").Changed {
		conf.Network.Port = viper.GetInt("network.port")
	}

	log.INFO.Printf("listening at :%d", conf.Network.Port)

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
	valueChan = make(chan util.Param)
	go tee.Run(valueChan)

	// setup environment
	if err == nil {
		err = configureEnvironment(conf)
	}

	// setup site and loadpoints
	var site *core.Site
	if err == nil {
		cp.TrackVisitors() // track duplicate usage
		site, err = configureSiteAndLoadpoints(conf)
	}

	// setup database
	if err == nil && conf.Influx.URL != "" {
		configureDatabase(conf.Influx, site.LoadPoints(), tee.Attach())
	}

	// setup mqtt publisher
	if err == nil && conf.Mqtt.Broker != "" {
		publisher := server.NewMQTT(conf.Mqtt.RootTopic())
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
		pushChan, err = configureMessengers(conf.Messaging, cache)
	}

	stopC := make(chan struct{})
	go shutdown.Run(stopC)

	siteC := make(chan struct{})

	// show main ui
	if err == nil {
		httpd.RegisterSiteHandlers(site, cache)

		// set channels
		site.DumpConfig()
		site.Prepare(valueChan, pushChan)

		// version check
		go updater.Run(log, httpd, tee, valueChan)

		// capture log messages for UI
		util.CaptureLogs(valueChan)

		// expose sponsor to UI
		if sponsor.Subject != "" {
			publish("sponsor", sponsor.Subject)
		}

		// allow web access for vehicles
		cp.webControl(conf.Network, httpd.Router(), valueChan)

		go func() {
			site.Run(stopC, conf.Interval)
			close(siteC)
		}()
	} else {
		var once sync.Once
		httpd.RegisterShutdownHandler(func() {
			once.Do(func() {
				log.FATAL.Println("evcc was stopped. OS should restart the service. Or restart manually.")
				close(siteC)
			})
		})

		// delayed reboot on error
		const rebootDelay = 5 * time.Minute

		log.FATAL.Println(err)
		log.FATAL.Printf("will attempt restart in: %v", rebootDelay)

		publishErrorInfo(cfgFile, err)

		go func() {
			select {
			case <-time.After(rebootDelay):
			case <-siteC:
			}
			os.Exit(1)
		}()
	}

	// uds health check listener
	go server.HealthListener(site, siteC)

	// catch signals
	go func() {
		signalC := make(chan os.Signal, 1)
		signal.Notify(signalC, os.Interrupt, syscall.SIGTERM)

		<-signalC    // wait for signal
		close(stopC) // signal loop to end

		exitC := make(chan struct{})
		wg := new(sync.WaitGroup)
		wg.Add(2)

		// wait for main loop and shutdown functions to finish
		go func() { <-shutdown.Done(conf.Interval); wg.Done() }()
		go func() { <-siteC; wg.Done() }()
		go func() { wg.Wait(); close(exitC) }()

		select {
		case <-exitC: // wait for loop to end
		case <-time.NewTimer(conf.Interval).C: // wait max 1 period
		}

		os.Exit(1)
	}()

	log.FATAL.Println(httpd.ListenAndServe())
}
