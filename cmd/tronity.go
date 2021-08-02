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
	"github.com/andig/evcc/vehicle/tronity"
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

// github.com/uhthomas/tesla
func state() string {
	var b [9]byte
	if _, err := io.ReadFull(rand.Reader, b[:]); err != nil {
		panic(err)
	}
	return base64.RawURLEncoding.EncodeToString(b[:])
}

func tronityToken(username, password string) {
	uri := tronity.OAuth2Config.AuthCodeURL(state(), oauth2.AccessTypeOffline)
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
	token, err := tronity.OAuth2Config.Exchange(ctx, code,
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
