package cmd

import (
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/server"
	"github.com/evcc-io/evcc/util"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/exp/slices"
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
	log.INFO.Printf("evcc %s", server.FormattedVersion())

	// load config
	if err := loadConfigFile(&conf); err != nil {
		log.FATAL.Fatal(err)
	}

	setLogLevel(cmd)

	var vehicleConf qualifiedConfig
	if len(conf.Vehicles) == 1 {
		vehicleConf = conf.Vehicles[0]
	} else if len(args) == 1 {
		idx := slices.IndexFunc(conf.Vehicles, func(v qualifiedConfig) bool {
			return strings.EqualFold(v.Name, args[0])
		})

		if idx >= 0 {
			vehicleConf = conf.Vehicles[idx]
		}
	}

	if vehicleConf.Name == "" {
		vehicles := lo.Map(conf.Vehicles, func(v qualifiedConfig, _ int) string {
			return v.Name
		})
		log.FATAL.Fatalf("vehicle not found, have %v", vehicles)
	}

	var token *oauth2.Token
	var err error

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
