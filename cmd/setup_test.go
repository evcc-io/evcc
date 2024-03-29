package cmd

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/util"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestYamlOff(t *testing.T) {
	var conf globalConfig
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(strings.NewReader(`loadpoints:
- mode: off
`)); err != nil {
		t.Error(err)
	}

	if err := viper.UnmarshalExact(&conf); err != nil {
		t.Error(err)
	}

	var lp core.Loadpoint
	if err := util.DecodeOther(conf.Loadpoints[0], &lp); err != nil {
		t.Error(err)
	}

	if lp.Mode_ != api.ModeOff {
		t.Errorf("expected `off`, got %s", lp.Mode_)
	}
}

func TestCircuitConf(t *testing.T) {
	var conf globalConfig
	viper.SetConfigType("yaml")

	require.NoError(t, viper.ReadConfig(strings.NewReader(`circuits:
- name: master
  maxPower: 10000
- name: slave
  parent: master
  maxPower: 10000
`)))

	require.NoError(t, viper.UnmarshalExact(&conf))

	cc, err := configureCircuits(conf)
	require.NoError(t, err)
	require.Len(t, cc, 2)
}
