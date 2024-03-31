package cmd

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/meter"
	"github.com/evcc-io/evcc/tariff"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
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
loadpoints:
- charger: test
  circuit: slave
`)))

	require.NoError(t, viper.UnmarshalExact(&conf))

	circuits, err := configureCircuits(conf)
	require.NoError(t, err)
	require.Len(t, circuits, 2)

	// empty charger
	require.NoError(t, config.Chargers().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Charger(nil))))

	lps, err := configureLoadpoints(conf, circuits)
	require.NoError(t, err)
	require.Len(t, lps, 1)
	require.NotNil(t, lps[0].GetCircuit())
}

func TestLoadpointMissingCircuitError(t *testing.T) {
	var conf globalConfig
	viper.SetConfigType("yaml")

	require.NoError(t, viper.ReadConfig(strings.NewReader(`
loadpoints:
- charger: test
`)))

	require.NoError(t, viper.UnmarshalExact(&conf))

	circuits := map[string]*core.Circuit{
		"master": new(core.Circuit),
	}

	// empty charger
	require.NoError(t, config.Chargers().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Charger(nil))))

	_, err := configureLoadpoints(conf, circuits)
	require.Error(t, err)
}

func TestSiteMissingCircuitError(t *testing.T) {
	var conf globalConfig
	viper.SetConfigType("yaml")

	require.NoError(t, viper.ReadConfig(strings.NewReader(`
loadpoints:
- charger: test
site:
  meters:
    grid: grid
`)))

	require.NoError(t, viper.UnmarshalExact(&conf))

	circuits := map[string]*core.Circuit{
		"master": new(core.Circuit),
	}

	lps := []*core.Loadpoint{
		new(core.Loadpoint),
	}

	// mock meter
	m, _ := meter.NewConfigurable(func() (float64, error) {
		return 0, nil
	})
	require.NoError(t, config.Meters().Add(config.NewStaticDevice(config.Named{
		Name: "grid",
	}, api.Meter(m))))

	// mock charger
	require.NoError(t, config.Chargers().Add(config.NewStaticDevice(config.Named{
		Name: "test",
	}, api.Charger(nil))))

	_, err := configureSite(conf.Site, lps, circuits, new(tariff.Tariffs))
	require.Error(t, err)
}
