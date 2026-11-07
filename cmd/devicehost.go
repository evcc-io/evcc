package cmd

import (
	"context"
	"errors"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util/auth"
	"github.com/evcc-io/evcc/util/config"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

const flagListen = "listen"

// devicehostCmd represents the devicehost command
var devicehostCmd = &cobra.Command{
	Use:   "devicehost",
	Short: "Host devices created and destroyed at runtime via the device config api (not persisted)",
	Run:   runDevicehost,
	// handle signals in runDevicehost for releasing devices
	PersistentPreRun: func(*cobra.Command, []string) {},
}

func init() {
	rootCmd.AddCommand(devicehostCmd)
	withCustomTemplate(devicehostCmd)

	devicehostCmd.Flags().String(flagListen, "127.0.0.1:7090", "Listen address")
}

func runDevicehost(cmd *cobra.Command, args []string) {
	// load config for environment (mqtt, network, ...) only
	if err := loadConfigFile(&conf, false); err != nil {
		log.FATAL.Fatal(err)
	}

	// devices live in memory only
	conf.Database.Type = "sqlite"
	conf.Database.Dsn = ":memory:"

	if err := configureEnvironment(cmd, &conf); err != nil {
		log.FATAL.Fatal(err)
	}

	site, err := core.NewSiteFromConfig(nil)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	addr := cmd.Flag(flagListen).Value.String()
	if host, _, err := net.SplitHostPort(addr); err != nil || !net.ParseIP(host).IsLoopback() {
		log.WARN.Println("❗❗❗ devicehost has no authentication, listening on non-loopback address", addr)
	}

	authObject := auth.New()
	authObject.SetAuthMode(auth.Disabled)

	router := mux.NewRouter()
	server.RegisterDeviceHandlers(router, site, authObject)

	srv := &http.Server{Addr: addr, Handler: router, ReadHeaderTimeout: 10 * time.Second}

	go func() {
		log.INFO.Println("devicehost listening at", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.FATAL.Fatal(err)
		}
	}()

	signalC := make(chan os.Signal, 1)
	signal.Notify(signalC, os.Interrupt, syscall.SIGTERM)
	<-signalC

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)

	// release all devices
	releaseAll(config.Meters())
	releaseAll(config.Chargers())
	releaseAll(config.Vehicles())
	releaseAll(config.Circuits())
	releaseAll(config.Curtailers())
	releaseAll(config.Tariffs())
	releaseAll(config.Messengers())
	releaseAll(config.Hems())
}

func releaseAll[T any](h config.Handler[T]) {
	for _, dev := range h.Devices() {
		if err := h.Delete(dev.Config().Name); err != nil {
			log.ERROR.Println(err)
		}
	}
}
