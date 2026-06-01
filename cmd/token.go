package cmd

import (
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/evcc-io/evcc/util"
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

	vehicles, err := tokenVehicles(args...)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	for _, vehicleConf := range vehicles {
		var token *oauth2.Token
		var err error

		isTemplate := strings.ToLower(vehicleConf.Type) == "template"
		if isTemplate {
			instance, err := renderTokenInstance(vehicleConf.Other)
			if err != nil {
				log.FATAL.Fatalf("rendering template failed: %v", err)
			}
			vehicleConf.Type = instance.Type
			vehicleConf.Other = instance.Other
		}

		typ := strings.ToLower(vehicleConf.Type)

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

// tokenVehicles returns the configured vehicles (static config and database),
// optionally filtered by name. Unlike configureVehicles it does not instantiate
// the vehicles: the whole point of this command is to generate the very
// credentials whose absence would make instantiation fail.
func tokenVehicles(names ...string) ([]config.Named, error) {
	var res []config.Named

	for _, cc := range conf.Vehicles {
		if len(names) == 0 || slices.Contains(names, cc.Name) {
			res = append(res, cc)
		}
	}

	configurable, err := config.ConfigurationsByClass(templates.Vehicle)
	if err != nil {
		return nil, err
	}

	for _, dev := range configurable {
		if cc := dev.Named(); len(names) == 0 || slices.Contains(names, cc.Name) {
			res = append(res, cc)
		}
	}

	if len(res) == 0 {
		if len(names) > 0 {
			return nil, fmt.Errorf("vehicle not found: %s", strings.Join(names, ", "))
		}
		return nil, errors.New("no vehicles configured")
	}

	return res, nil
}

// renderTokenInstance renders a vehicle template instance, tolerating the missing
// required credentials (the access/refresh tokens this command generates). When
// the regular render fails validation, empty required params are stubbed so the
// instance type can still be resolved. The stubbed params are exactly the empty
// credentials, which the token generators do not read.
func renderTokenInstance(other map[string]any) (*templates.Instance, error) {
	instance, err := templates.RenderInstance(templates.Vehicle, other)
	if err == nil {
		return instance, nil
	}

	// only tolerate config (validation) errors, e.g. a missing required token
	if _, ok := errors.AsType[*util.ConfigError](err); !ok {
		return nil, err
	}

	var cc struct {
		Template string
		Other    map[string]any `mapstructure:",remain"`
	}
	if derr := util.DecodeOther(other, &cc); derr != nil {
		return nil, err
	}

	tmpl, terr := templates.ByName(templates.Vehicle, cc.Template)
	if terr != nil {
		return nil, err
	}

	stub := maps.Clone(other)
	for i := range tmpl.Params {
		if p := &tmpl.Params[i]; p.IsRequired() {
			if v, ok := stub[p.Name]; !ok || v == nil || v == "" {
				stub[p.Name] = "unset"
			}
		}
	}

	return templates.RenderInstance(templates.Vehicle, stub)
}
