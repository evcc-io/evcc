package cmd

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/server/modbus"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
)

func TestYamlOff(t *testing.T) {
	var conf globalconfig.All
	var lp core.Loadpoint
	viper.SetConfigType("yaml")

	require.NoError(t, viper.ReadConfig(strings.NewReader("loadpoints:\n- mode: off")))
	require.NoError(t, viper.UnmarshalExact(&conf))
	require.NoError(t, util.DecodeOther(conf.Loadpoints[0].Other, &lp))
	require.Equal(t, api.ModeOff, lp.DefaultMode)
}

func TestYamlModbusProxyReadonlyTrue(t *testing.T) {
	var conf globalconfig.All
	viper.SetConfigType("yaml")

	require.NoError(t, viper.ReadConfig(strings.NewReader("modbusproxy:\n- readonly: true")))
	require.NoError(t, viper.UnmarshalExact(&conf))
	require.Equal(t, modbus.ReadOnlyTrue, conf.ModbusProxy[0].ReadOnly)
}
