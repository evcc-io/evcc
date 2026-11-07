package cmd

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util/auth"
	"github.com/evcc-io/evcc/util/config"
	"github.com/gorilla/mux"
	"github.com/spf13/cobra"
)

const (
	flagListen = "listen"
	flagToken  = "token"
)

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
	devicehostCmd.Flags().String(flagToken, "", "Bearer token required for api requests (default random)")
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

	// bearer token protects against browser-based requests (CSRF, DNS rebinding)
	token := cmd.Flag(flagToken).Value.String()
	if token == "" {
		b := make([]byte, 16)
		_, _ = rand.Read(b)
		token = hex.EncodeToString(b)
		log.INFO.Println("devicehost token:", token)
	}

	// token replaces evcc's session auth
	authObject := auth.New()
	authObject.SetAuthMode(auth.Disabled)

	router := mux.NewRouter()
	router.Use(bearerAuth(token))
	server.RegisterDeviceHandlers(router, site, authObject)

	srv := &http.Server{
		Addr:              addr,
		Handler:           http.MaxBytesHandler(router, 1<<20),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       5 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

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
	if err := srv.Shutdown(ctx); err != nil {
		_ = srv.Close()
	}

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
	// clone since Delete modifies the slice
	for _, dev := range slices.Clone(h.Devices()) {
		if err := h.Delete(dev.Config().Name); err != nil {
			log.ERROR.Println(err)
		}
	}
}

func bearerAuth(token string) mux.MiddlewareFunc {
	want := []byte("Bearer " + token)
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if subtle.ConstantTimeCompare([]byte(r.Header.Get("Authorization")), want) != 1 {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
