package cmd

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
	"golang.org/x/oauth2"
)

// tokenCmd represents the vehicle command
var tokenCmd = &cobra.Command{
	Use:   "token [vehicle name]",
	Short: "Generate token credentials",
	Run:   runToken,
}

func init() {
	rootCmd.AddCommand(tokenCmd)
}

func runToken(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config
	conf, err := loadConfigFile(cfgFile)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	var vehicleConf qualifiedConfig
	if len(conf.Vehicles) == 1 {
		vehicleConf = conf.Vehicles[0]
	} else if len(args) == 1 {
		vehicleConf = funk.Find(conf.Vehicles, func(v qualifiedConfig) bool {
			return strings.EqualFold(v.Name, args[0])
		}).(qualifiedConfig)
	}

	if vehicleConf.Name == "" {
		vehicles := funk.Map(conf.Vehicles, func(v qualifiedConfig) string {
			return v.Name
		}).([]string)
		log.FATAL.Fatalf("vehicle not found, have %v", vehicles)
	}

	var token *oauth2.Token

	switch strings.ToLower(vehicleConf.Type) {
	case "tesla":
		token, err = teslaToken()
	case "tronity":
		token, err = tronityToken(conf, vehicleConf)
	default:
		log.FATAL.Fatalf("vehicle type '%s' does not support token authentication", vehicleConf.Type)
	}

	if err != nil {
		log.FATAL.Fatal(err)
	}

	fmt.Println()
	fmt.Println("Add the following tokens to the vehicle config:")
	fmt.Println()
	fmt.Println("  tokens:")
	fmt.Println("    access:", token.AccessToken)
	fmt.Println("    refresh:", token.RefreshToken)
}
