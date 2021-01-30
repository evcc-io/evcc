package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/andig/evcc/server"
	"github.com/andig/evcc/util"
	mfa "github.com/andig/evcc/vehicle/tesla"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/thoas/go-funk"
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

func generateToken(user, pass string) {
	client, err := mfa.NewClient()
	if err != nil {
		panic(err)
	}

	passcodeC := make(chan string, 1)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		ctx := context.Background()
		token, err := client.Login(ctx, user, pass, passcodeC)
		if err != nil {
			panic(err)
		}

		fmt.Println(token)
		wg.Done()
	}()

	fmt.Print("Please enter passcode: ")
	reader := bufio.NewReader(os.Stdin)
	code, _ := reader.ReadString('\n')
	passcodeC <- strings.TrimSpace(code)

	wg.Wait()
}

func runTeslaToken(cmd *cobra.Command, args []string) {
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
			return strings.ToLower(v.Name) == strings.ToLower(args[0])
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
