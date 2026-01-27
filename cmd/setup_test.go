package cmd

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/util"
	vpr "github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

func TestYamlOff(t *testing.T) {
	var conf globalconfig.All
	viper := vpr.NewWithOptions(vpr.ExperimentalBindStruct())
	viper.SetConfigType("yaml")

	require.NoError(t, viper.ReadConfig(strings.NewReader("loadpoints:\n- mode: off")))
	require.NoError(t, viper.UnmarshalExact(&conf))

	var lp core.Loadpoint
	require.NoError(t, util.DecodeOther(conf.Loadpoints[0].Other, &lp))
	require.Equal(t, api.ModeOff, lp.DefaultMode)
}
