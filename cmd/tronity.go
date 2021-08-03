package cmd

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/vehicle"
	"github.com/andig/evcc/vehicle/tronity"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
	"golang.org/x/oauth2"
)

// tronityCmd represents the vehicle command
var tronityCmd = &cobra.Command{
	Use:   "tronity-token",
	Short: "Generate Tronity token credentials",
	Run:   runTronityToken,
}

func init() {
	rootCmd.AddCommand(tronityCmd)
}

// github.com/uhthomas/tesla
func state() string {
	var b [9]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func tokenExchangeHandler(oc *oauth2.Config, state string, resC chan *oauth2.Token) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if remote := r.URL.Query().Get("state"); state != remote {
			w.WriteHeader(http.StatusBadRequest)
			resC <- nil
			return
		}

		code := r.URL.Query().Get("code")

		ctx := context.Background()
		token, err := oc.Exchange(ctx, code,
			oauth2.SetAuthURLParam("grant_type", "code"), // app
		)

		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			fmt.Fprintln(w, err)
			resC <- nil
			return
		}

		fmt.Fprintln(w, "Token received, see console")
		resC <- token
	}
}

func tronityToken(addr string, oc *oauth2.Config) error {
	state := state()

	uri := oc.AuthCodeURL(state, oauth2.AccessTypeOffline)
	uri = strings.ReplaceAll(uri, "scope=", "scopes=")

	if err := open.Start(uri); err != nil {
		return err
	}

	resC := make(chan *oauth2.Token)
	handler := tokenExchangeHandler(oc, state, resC)
	defer close(resC)

	// handle request
	mux := &http.ServeMux{}
	mux.HandleFunc("/auth/tronity", handler)

	wg := new(sync.WaitGroup)
	s := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	// start server
	wg.Add(1)
	go func() {
		if err := s.ListenAndServe(); err != http.ErrServerClosed {
			log.FATAL.Fatal(err)
		}
		wg.Done()
	}()

	// close on exit
	defer func() {
		_ = s.Close()
		wg.Wait()
		close(resC)
	}()

	t := time.NewTimer(time.Minute)

	select {
	case <-t.C:
		return errors.New("timeout")

	case token := <-resC:
		if token == nil {
			return errors.New("token not received")
		}

		fmt.Println()
		fmt.Println("Add the following tokens to the tronity vehicle config:")
		fmt.Println()
		fmt.Println("  tokens:")
		fmt.Println("    access:", token.AccessToken)
		fmt.Println("    refresh:", token.RefreshToken)

		return nil
	}
}

func runTronityToken(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config
	conf, err := loadConfigFile(cfgFile)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	// find vehicle by type and name
	vehicles := funk.Filter(conf.Vehicles, func(v qualifiedConfig) bool {
		return strings.ToLower(v.Type) == "tronity"
	}).([]qualifiedConfig)

	var vehicleConf qualifiedConfig
	if len(vehicles) == 1 {
		vehicleConf = vehicles[0]
	} else if len(args) == 1 {
		vehicleConf = funk.Find(vehicles, func(v qualifiedConfig) bool {
			return strings.EqualFold(v.Name, args[0])
		}).(qualifiedConfig)
	}

	if vehicleConf.Name == "" {
		log.FATAL.Fatal("vehicle not found")
	}

	cc := struct {
		Client      vehicle.ClientCredentials
		RedirectURI string
		Other       map[string]interface{} `mapstructure:",remain"`
	}{}

	if err := util.DecodeOther(vehicleConf.Other, &cc); err != nil {
		log.FATAL.Fatal(err)
	}

	oc, err := tronity.OAuth2Config(cc.Client.ID, cc.Client.Secret)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	if oc.RedirectURL = cc.RedirectURI; oc.RedirectURL == "" {
		_, port, err := net.SplitHostPort(conf.URI)
		if err != nil {
			log.FATAL.Fatal(err)
		}

		oc.RedirectURL = fmt.Sprintf("http://%s/auth/tronity", net.JoinHostPort("localhost", port))
	}

	if err := tronityToken(conf.URI, oc); err != nil {
		log.FATAL.Fatal(err)
	}
}
