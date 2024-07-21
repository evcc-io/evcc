package cmd

import (
	_ "embed" // for yaml
	"fmt"
	"strings"

	"github.com/evcc-io/evcc/api/globalconfig"
)

//go:embed demo.yaml
var demoYaml string

func demoConfig(conf *globalconfig.All) error {
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(strings.NewReader(demoYaml)); err != nil {
		return fmt.Errorf("failed decoding demo config: %w", err)
	}

	if err := viper.UnmarshalExact(conf); err != nil {
		return fmt.Errorf("failed loading demo config: %w", err)
	}

	// parse log levels after reading config
	parseLogLevels()

	return nil
}
