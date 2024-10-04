package loadpoint

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
)

type StaticConfig struct {
	// static config
	Charger string `json:"charger,omitempty"`
	Meter   string `json:"meter,omitempty"`
	Circuit string `json:"circuit,omitempty"`
	Vehicle string `json:"vehicle,omitempty"`
}

type DynamicConfig struct {
	// dynamic config
	Title          string   `json:"title"`
	Mode           string   `json:"mode"`
	Priority       int      `json:"priority"`
	Phases         int      `json:"phases"`
	MinCurrent     float64  `json:"minCurrent"`
	MaxCurrent     float64  `json:"maxCurrent"`
	SmartCostLimit *float64 `json:"smartCostLimit"`

	Thresholds ThresholdsConfig `json:"thresholds"`
	Soc        SocConfig        `json:"soc"`
}

func SplitConfig(payload map[string]any) (DynamicConfig, map[string]any, error) {
	// split static and dynamic config via mapstructure
	var cc struct {
		DynamicConfig `mapstructure:",squash"`
		Other         map[string]any `mapstructure:",remain"`
	}

	if err := util.DecodeOther(payload, &cc); err != nil {
		return DynamicConfig{}, nil, err
	}

	// TODO: proper handling of id/name
	delete(cc.Other, "id")
	delete(cc.Other, "name")

	return cc.DynamicConfig, cc.Other, nil
}

func (payload DynamicConfig) Apply(lp API) error {
	lp.SetTitle(payload.Title)
	lp.SetPriority(payload.Priority)
	lp.SetSmartCostLimit(payload.SmartCostLimit)
	lp.SetThresholds(payload.Thresholds)

	// TODO mode warning
	lp.SetSocConfig(payload.Soc)

	mode, err := api.ChargeModeString(payload.Mode)
	if err == nil {
		lp.SetMode(mode)
	}

	if err == nil {
		err = lp.SetPhases(payload.Phases)
	}

	if err == nil {
		err = lp.SetMinCurrent(payload.MinCurrent)
	}

	if err == nil {
		err = lp.SetMaxCurrent(payload.MaxCurrent)
	}

	return err
}
