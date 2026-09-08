package cmd

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
	"github.com/stretchr/testify/require"
)

func TestYamlOff(t *testing.T) {
	var conf globalconfig.All
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
	if err := util.DecodeOther(conf.Loadpoints[0].Other, &lp); err != nil {
		t.Error(err)
	}

	if lp.DefaultMode != api.ModeOff {
		t.Errorf("expected `off`, got %s", lp.DefaultMode)
	}
}

type configCompleteCharger struct {
	api.Charger
	complete bool
}

func (c *configCompleteCharger) ConfigComplete() {
	c.complete = true
}

func TestCompleteChargerConfiguration(t *testing.T) {
	config.Reset()
	t.Cleanup(config.Reset)

	charger := &configCompleteCharger{}
	require.NoError(t, config.Chargers().Add(config.NewStaticDevice(config.Named{Name: "test"}, api.Charger(charger))))

	completeChargerConfiguration()

	require.True(t, charger.complete)
}
