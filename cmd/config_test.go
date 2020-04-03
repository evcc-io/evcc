package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/spf13/viper"
)

func TestMeterConfig(t *testing.T) {
	yaml := `
meters:
- name: pv
  type: exec # mqtt will fail due to missing global
  power: topic
- name: charge
  type: exec
  power: script
`
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(bytes.NewBuffer([]byte(yaml))); err != nil {
		t.Error(err)
	}
}
func TestChargerConfig(t *testing.T) {
	yaml := `
chargers:
- name: test
  type: configurable
  status:
    type: script
    cmd: script
  enable:
    type: script
    cmd: script
  enabled:
    type: script
    cmd: script
  maxCurrent:
    type: script
    cmd: script
- name: wallbe
  type: wallbe
  uri: 192.168.0.8:502
`
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(bytes.NewBuffer([]byte(yaml))); err != nil {
		t.Error(err)
	}
}

func TestDistConfig(t *testing.T) {
	file := "../evcc.dist.yaml"
	yaml, err := os.Open(file)
	if err != nil {
		t.Error(err)
	}

	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(yaml); err != nil {
		t.Error(err)
	}

	// check config does not contain surplus keys
	var conf config
	if err := viper.UnmarshalExact(&conf); err != nil {
		log.FATAL.Fatalf("config: failed parsing config file %s: %v", cfgFile, err)
	}

	// check config is valid
	loadConfig(conf, nil)
}
