package cmd

import (
	"fmt"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util/config"
	"github.com/evcc-io/evcc/util/templates"
	"github.com/spf13/cobra"
	"golang.org/x/oauth2"
)

// tokenCmd represents the vehicle command
var tokenCmd = &cobra.Command{
	Use:   "token [vehicle name]",
	Short: "Generate token credentials",
	Run:   runToken,
}

// psaBrands lists the PSA group brands whose token flow needs only the brand
// name — no rendered vehicle config — so it can run before any tokens exist.
var psaBrands = []string{"citroen", "ds", "opel", "peugeot"}

func init() {
	rootCmd.AddCommand(tokenCmd)
}

func runToken(cmd *cobra.Command, args []string) {
	// load config
	if err := loadConfigFile(&conf, !cmd.Flag(flagIgnoreDatabase).Changed); err != nil {
		log.FATAL.Fatal(err)
	}

	// setup environment
	if err := configureEnvironment(cmd, &conf); err != nil {
		log.FATAL.Fatal(err)
	}

	if len(args) == 0 {
		log.FATAL.Fatal("vehicle name required")
	}

	// resolve vehicle configs directly from YAML — configureVehicles would
	// instantiate them and fail validation on the very accessToken/refreshToken
	// fields we are about to generate (evcc-io/evcc#29864).
	var targets []config.Named
	for _, cc := range conf.Vehicles {
		if slices.Contains(args, cc.Name) {
			targets = append(targets, cc)
		}
	}
	if len(targets) == 0 {
		log.FATAL.Fatalf("no matching vehicle in config: %v", args)
	}

	for _, vehicleConf := range targets {
		var token *oauth2.Token
		var err error

		isTemplate := strings.ToLower(vehicleConf.Type) == "template"
		typ := strings.ToLower(vehicleConf.Type)

		if isTemplate {
			tplName, _ := vehicleConf.Other["template"].(string)
			tplLower := strings.ToLower(tplName)

			// PSA token flow needs only the brand — skip template rendering so
			// that missing accessToken/refreshToken (the values we are about
			// to create) don't trip required-field validation.
			if slices.Contains(psaBrands, tplLower) {
				typ = tplLower
			} else {
				instance, errR := templates.RenderInstance(templates.Vehicle, vehicleConf.Other)
				if errR != nil {
					log.FATAL.Fatalf("rendering template failed: %v", errR)
				}
				vehicleConf.Type = instance.Type
				vehicleConf.Other = instance.Other
				typ = strings.ToLower(instance.Type)
			}
		}

		switch typ {
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
}
