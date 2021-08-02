package cmd

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	"github.com/skratchdot/open-golang/open"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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

// OAuth2Config is the OAuth2 configuration for authenticating with the Tesla API.
var OAuth2Config = &oauth2.Config{
	ClientID:    "db23e992-c64a-4263-9a9e-8f8f1b46ec41",
	RedirectURL: "http://localhost:8080",
	Endpoint: oauth2.Endpoint{
		AuthURL:  "https://api-eu.tronity.io/oauth/authorize",
		TokenURL: "https://api-eu.tronity.io/oauth/authentication",
	},
	Scopes: []string{"read_vin", "read_vehicle_info", "read_odometer", "read_charge", "read_charge", "read_battery", "read_location", "write_charge_start_stop", "write_wake_up"},
}

// github.com/uhthomas/tesla
func state() string {
	var b [9]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func tronityToken(username, password string) {
	uri := OAuth2Config.AuthCodeURL(state(), oauth2.AccessTypeOffline)
	uri = strings.ReplaceAll(uri, "scope=", "scopes=")
	if err := open.Start(uri); err != nil {
		panic(err)
	}

	// fmt.Println()
	// fmt.Println("Add the following tokens to the tesla vehicle config:")
	// fmt.Println()
	// fmt.Println("  tokens:")
	// fmt.Println("    access:", token.AccessToken)
	// fmt.Println("    refresh:", token.RefreshToken)
}

func handleMain(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")
	fmt.Println(code)
	fmt.Println(state)

	ctx := context.Background()
	token, err := OAuth2Config.Exchange(ctx, code,
		oauth2.SetAuthURLParam("grant_type", "code"),
	)

	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, err)
		return
	}

	fmt.Fprintf(w, "%+v", token)
}

func runTronityToken(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	mux := &http.ServeMux{}
	mux.HandleFunc("/", handleMain)

	s := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}
	go func() { log.FATAL.Fatal(s.ListenAndServe()) }()

	user := ""
	password := ""
	tronityToken(user, password)

	time.Sleep(time.Minute)
}
