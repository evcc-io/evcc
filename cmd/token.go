package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util/config"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
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
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	var vehicleConf config.Named
	if len(conf.Vehicles) == 1 {
		vehicleConf = conf.Vehicles[0]
	} else if len(args) == 1 {
		idx := slices.IndexFunc(conf.Vehicles, func(v config.Named) bool {
			return strings.EqualFold(v.Name, args[0])
		})

		if idx >= 0 {
			vehicleConf = conf.Vehicles[idx]
		}
	}

	if vehicleConf.Name == "" {
		vehicles := lo.Map(conf.Vehicles, func(v config.Named, _ int) string {
			return v.Name
		})
		log.FATAL.Fatalf("vehicle not found, have %v", vehicles)
	}

	var token *oauth2.Token
	var err error

	typ := strings.ToLower(vehicleConf.Type)
	if typ == "template" {
		typ = strings.ToLower(vehicleConf.Other["template"].(string))
	}

	switch typ {
	case "mercedes":
		token, err = mercedesToken()
	case "tronity":
		token, err = tronityToken(conf, vehicleConf)
	case "citroen", "ds", "opel", "peugeot":
		token, err = psaToken(typ)

	default:
		log.FATAL.Fatalf("vehicle type '%s' does not support token authentication", vehicleConf.Type)
	}

	if err != nil {
		log.FATAL.Fatal(err)
	}

	fmt.Println()
	fmt.Println("Add the following tokens to the vehicle config:")
	fmt.Println()
	fmt.Println("    tokens:")
	fmt.Println("      access:", token.AccessToken)
	fmt.Println("      refresh:", token.RefreshToken)
}
