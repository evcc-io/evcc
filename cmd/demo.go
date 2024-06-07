package cmd

import (
	_ "embed" // for yaml
	"fmt"
	"strings"
)

//go:embed demo.yaml
var demoYaml string

func demoConfig(conf *globalConfig) error {
	vpr.SetConfigType("yaml")
	if err := vpr.ReadConfig(strings.NewReader(demoYaml)); err != nil {
		return fmt.Errorf("failed decoding demo config: %w", err)
	}

	if err := vpr.UnmarshalExact(conf); err != nil {
		return fmt.Errorf("failed loading demo config: %w", err)
	}

	// parse log levels after reading config
	parseLogLevels()

	return nil
}
