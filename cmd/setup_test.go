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

func TestWrapEEBusStartError(t *testing.T) {
	// realistic error from issue https://github.com/evcc-io/evcc/issues/31043
	skiErr := fmt.Errorf("%w (subject: CN=EVCC_HEMS_01,OU=,O=EVCC,C=DE, SKI: 0bae4a2cd7b1be3bc0f9ad285bc99a8919572574, expected: f337e022eb15020eeddee7bcafebabecafebabec)",
		shipapi.ErrInvalidSKI)

	tests := []struct {
		name     string
		err      error
		wantHint bool
	}{
		{"SKI mismatch (ship-go wrapped)", skiErr, true},
		{"unrelated error", errors.New("port already in use"), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := wrapEEBusStartError(tt.err)
			if got == nil {
				t.Fatal("expected non-nil error")
			}
			if !errors.Is(got, tt.err) {
				t.Errorf("wrapped error does not preserve underlying cause: %v", got)
			}
			hasHint := strings.Contains(got.Error(), "invalid Subject Key Identifier")
			if hasHint != tt.wantHint {
				t.Errorf("hint present = %v, want %v; error was:\n%s", hasHint, tt.wantHint, got)
			}
		})
	}
}
