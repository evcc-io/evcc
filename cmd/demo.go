package cmd

import (
	_ "embed" // for yaml
	"fmt"

	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
)

//go:embed demo.yaml
var demoYaml string

func demoConfig(conf *config) error {
	demo := map[string]interface{}{}

	if err := yaml.Unmarshal([]byte(demoYaml), &demo); err != nil {
		return fmt.Errorf("failed decoding demo config: %+w", err)
	}

	for k, v := range demo {
		viper.Set(k, v)
	}

	if err := viper.UnmarshalExact(&conf); err != nil {
		return fmt.Errorf("failed loading demo config: %w", err)
	}

	conf.Network.Port = defaultPort

	// parse log levels after reading config
	parseLogLevels()

	return nil
}
