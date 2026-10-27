package cmd

import (
	"strings"
	"testing"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/globalconfig"
	"github.com/evcc-io/evcc/core"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/config"
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

func TestYamlBatteryMasterForwardReference(t *testing.T) {
	var conf globalconfig.All
	viper.SetConfigType("yaml")
	if err := viper.ReadConfig(strings.NewReader(`meters:
- name: slave
  type: template
  master: battery
- name: battery
  type: template
  usage: battery
`)); err != nil {
		t.Fatal(err)
	}
	if err := viper.UnmarshalExact(&conf); err != nil {
		t.Fatal(err)
	}

	normalizeBatteryMasters(combineBatteryMeterConfigs(conf.Meters, nil), []string{"slave", "battery"})
	if got := conf.Meters[0].Other["master"]; got != "battery" {
		t.Fatalf("forward reference was not retained: got %v", got)
	}
}

func TestNormalizeBatteryMasters(t *testing.T) {
	static := []config.Named{
		{Name: "slave", Other: map[string]any{"master": "later"}},
		{Name: "later", Other: map[string]any{"usage": "battery"}},
		{Name: "db-slave", Other: map[string]any{"master": "db:42"}},
		{Name: "invalid", Other: map[string]any{"master": "missing"}},
		{Name: "self", Other: map[string]any{"master": "self"}},
		{Name: "empty", Other: map[string]any{"master": ""}},
		{Name: "unassigned-master", Other: map[string]any{"master": "unassigned"}},
		{Name: "unassigned", Other: map[string]any{"usage": "battery"}},
		{Name: "nested-slave", Other: map[string]any{"master": "nested-master"}},
		{Name: "nested-master", Other: map[string]any{"master": "later"}},
	}
	database := []config.Config{
		{ID: 42, Data: map[string]any{"usage": "battery"}},
		{ID: 43, Data: map[string]any{"master": "later"}},
		{ID: 44, Data: map[string]any{"master": "missing"}},
		{ID: 45, Data: map[string]any{"master": "db:42"}},
		{ID: 46, Data: map[string]any{"master": "db:45"}},
	}

	meters := combineBatteryMeterConfigs(static, database)
	warnings := normalizeBatteryMasters(meters, []string{"slave", "later", "db-slave", "invalid", "self", "empty", "unassigned-master", "nested-slave", "nested-master", "db:42", "db:43", "db:44", "db:45", "db:46"})

	if got := static[0].Other["master"]; got != "later" {
		t.Fatalf("forward reference was not retained: got %v", got)
	}
	if got := static[2].Other["master"]; got != "db:42" {
		t.Fatalf("database reference was not retained: got %v", got)
	}
	if got := static[3].Other["master"]; got != nil {
		t.Fatalf("invalid reference was not cleared: got %v", got)
	}
	if got := static[4].Other["master"]; got != nil {
		t.Fatalf("self reference was not cleared: got %v", got)
	}
	if got := static[5].Other["master"]; got != "" {
		t.Fatalf("empty reference was changed: got %v", got)
	}
	if got := static[6].Other["master"]; got != nil {
		t.Fatalf("unassigned battery was accepted as master: got %v", got)
	}
	if got := static[8].Other["master"]; got != nil {
		t.Fatalf("nested slave was accepted: got %v", got)
	}
	if got := static[9].Other["master"]; got != "later" {
		t.Fatalf("direct slave assignment was not retained: got %v", got)
	}
	if got := database[1].Data["master"]; got != "later" {
		t.Fatalf("valid UI reference was not retained: got %v", got)
	}
	if got := database[2].Data["master"]; got != nil {
		t.Fatalf("invalid UI reference was not cleared: got %v", got)
	}
	if got := database[3].Data["master"]; got != "db:42" {
		t.Fatalf("direct UI slave assignment was not retained: got %v", got)
	}
	if got := database[4].Data["master"]; got != nil {
		t.Fatalf("nested UI slave assignment was not cleared: got %v", got)
	}
	wantWarnings := map[string]struct {
		master string
		reason string
	}{
		"invalid":           {master: "missing", reason: batteryConfigWarningInvalidMaster},
		"self":              {master: "self", reason: batteryConfigWarningInvalidMaster},
		"unassigned-master": {master: "unassigned", reason: batteryConfigWarningInvalidMaster},
		"db:44":             {master: "missing", reason: batteryConfigWarningInvalidMaster},
		"nested-slave":      {master: "nested-master", reason: batteryConfigWarningMasterIsSlave},
		"db:46":             {master: "db:45", reason: batteryConfigWarningMasterIsSlave},
	}
	if len(warnings) != len(wantWarnings) {
		t.Fatalf("warning count: got %d, want %d: %+v", len(warnings), len(wantWarnings), warnings)
	}
	for _, warning := range warnings {
		if want, ok := wantWarnings[warning.Device]; !ok || warning.Master != want.master || warning.Reason != want.reason {
			t.Errorf("unexpected warning: %+v", warning)
		}
		delete(wantWarnings, warning.Device)
	}
	if len(wantWarnings) != 0 {
		t.Errorf("missing warnings: %v", wantWarnings)
	}
}
