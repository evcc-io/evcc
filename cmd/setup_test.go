package cmd

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	shipapi "github.com/enbility/ship-go/api"
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/util"
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

func TestEEBusSKIMismatchDetection(t *testing.T) {
	// realistic error string produced by ship-go's cert.SkiFromCertificate
	shipGoErr := fmt.Errorf("%w (subject: CN=EVCC_HEMS_01,OU=,O=EVCC,C=DE, SKI: 0bae4a2cd7b1be3bc0f9ad285bc99a8919572574, expected: f337e022eb15020eeddee7bcafebabecafebabec)",
		shipapi.ErrInvalidSKI)

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil error", nil, false},
		{"unrelated error", errors.New("something went wrong"), false},
		{"ship-go wrapped invalid SKI", shipGoErr, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := errors.Is(tt.err, shipapi.ErrInvalidSKI); got != tt.want {
				t.Errorf("errors.Is(%v, ErrInvalidSKI) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}
