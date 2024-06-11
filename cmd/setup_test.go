package cmd

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/util"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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
	if err := util.DecodeOther(conf.Loadpoints[0], &lp); err != nil {
		t.Error(err)
	}

	if lp.Mode_ != api.ModeOff {
		t.Errorf("expected `off`, got %s", lp.Mode_)
	}
}

func TestEEbusConfig(t *testing.T) {
	conf := `
eebus:
  certificate:
    private: |
      -----BEGIN EC PRIVATE KEY-----
      MHcCfoo==
      -----END EC PRIVATE KEY-----
    public: |
      -----BEGIN CERTIFICATE-----
      MIIBbar=
      -----END CERTIFICATE-----
`

	var res globalconfig.All
	err := yaml.Unmarshal([]byte(conf), &res)
	require.NoError(t, err)
}
