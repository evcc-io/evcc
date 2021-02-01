package cmd

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

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

func codeprompt() (string, error) {
	passcodeC := make(chan string)

	go func() {
		fmt.Print("Please enter passcode: ")
		reader := bufio.NewReader(os.Stdin)
		if code, err := reader.ReadString('\n'); err == nil {
			passcodeC <- strings.TrimSpace(code)
		}
	}()

	select {
	case <-time.NewTimer(30 * time.Second).C:
		return "", errors.New("code request expired")
	case code := <-passcodeC:
		return code, nil
	}
}

func generateToken(user, pass string) {
	client, err := mfa.NewClient(log)
	if err != nil {
		log.FATAL.Fatalln(err)
	}

	ctx := context.Background()
	token, err := client.Login(ctx, user, pass, codeprompt)
	if err != nil {
		log.FATAL.Fatalln(err)
	}

	fmt.Println(token)
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
