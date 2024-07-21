package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
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

	isTemplate := strings.ToLower(vehicleConf.Type) == "template"
	if isTemplate {
		instance, err := templates.RenderInstance(templates.Vehicle, vehicleConf.Other)
		if err != nil {
			log.FATAL.Fatalf("rendering template failed: %v", err)
		}
		vehicleConf.Type = instance.Type
		vehicleConf.Other = instance.Other
	}

	typ := strings.ToLower(vehicleConf.Type)

	switch typ {
	case "mercedes":
		token, err = mercedesToken()
	case "ford", "ford-connect":
		token, err = fordConnectToken(vehicleConf)
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

	if isTemplate {
		fmt.Println("    type: template")
		fmt.Println("    template:", typ)
		fmt.Println("    accesstoken:", token.AccessToken)
		fmt.Println("    refreshtoken:", token.RefreshToken)
	} else {
		fmt.Println("    type:", typ)
		fmt.Println("    tokens:")
		fmt.Println("      access:", token.AccessToken)
		fmt.Println("      refresh:", token.RefreshToken)
	}
}
