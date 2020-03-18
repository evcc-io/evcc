package cmd

import (
	"bytes"
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
