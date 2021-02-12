package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	auth "github.com/andig/evcc/vehicle/tesla"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
	"github.com/uhthomas/tesla"
)

// teslaCmd represents the vehicle command
var teslaCmd = &cobra.Command{
	Use:   "tesla-token [name]",
	Short: "Generate Tesla access token for configured vehicle",
	Run:   runTeslaToken,
}

func init() {
	rootCmd.AddCommand(teslaCmd)
}

func codePrompt(ctx context.Context, devices []tesla.Device) (tesla.Device, string, error) {
	fmt.Println("Authentication devices:", funk.Map(devices, func(d tesla.Device) string {
		return d.Name
	}))
	if len(devices) > 1 {
		return tesla.Device{}, "", errors.New("multiple devices found, only single device supported")
	}

	fmt.Print("Please enter passcode: ")
	reader := bufio.NewReader(os.Stdin)
	code, err := reader.ReadString('\n')

	return devices[0], strings.TrimSpace(code), err
}

func generateToken(user, pass string) {
	client, err := auth.NewClient(log)
	if err != nil {
		log.FATAL.Fatalln(err)
	}

	client.DeviceHandler(codePrompt)

	token, err := client.Login(user, pass)
	if err != nil {
		log.FATAL.Fatalln(err)
	}

	fmt.Println()
	fmt.Println("Add the following tokens to the tesla vehicle config:")
	fmt.Println()
	fmt.Println("  tokens:")
	fmt.Println("    access:", token.AccessToken)
	fmt.Println("    refresh:", token.RefreshToken)
}

func runTeslaToken(cmd *cobra.Command, args []string) {
	util.LogLevel(viper.GetString("log"), viper.GetStringMapString("levels"))
	log.INFO.Printf("evcc %s (%s)", server.Version, server.Commit)

	// load config
	conf, err := loadConfigFile(cfgFile)
	if err != nil {
		log.FATAL.Fatal(err)
	}

	teslas := funk.Filter(conf.Vehicles, func(v qualifiedConfig) bool {
		return strings.ToLower(v.Type) == "tesla"
	}).([]qualifiedConfig)

	var vehicleConf qualifiedConfig
	if len(teslas) == 1 {
		vehicleConf = teslas[0]
	} else if len(args) == 1 {
		vehicleConf = funk.Find(teslas, func(v qualifiedConfig) bool {
			return strings.EqualFold(v.Name, args[0])
		}).(qualifiedConfig)
	}

	if vehicleConf.Name == "" {
		log.FATAL.Fatal("vehicle not found")
	}

	var credentials struct {
		User, Password string
		Other          map[string]interface{} `mapstructure:",remain"`
	}

	if err := util.DecodeOther(vehicleConf.Other, &credentials); err != nil {
		log.FATAL.Fatal(err)
	}

	generateToken(credentials.User, credentials.Password)
}
