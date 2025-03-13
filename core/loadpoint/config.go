package loadpoint

import (
	"time"

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
	Title            string    `json:"title"`
	DefaultMode      string    `json:"defaultMode"`
	Priority         int       `json:"priority"`
	PhasesConfigured int       `json:"phasesConfigured"`
	MinCurrent       float64   `json:"minCurrent"`
	MaxCurrent       float64   `json:"maxCurrent"`
	SmartCostLimit   *float64  `json:"smartCostLimit"`
	PlanEnergy       float64   `json:"planEnergy"`
	PlanTime         time.Time `json:"planTime"`
	LimitEnergy      float64   `json:"limitEnergy"`
	LimitSoc         int       `json:"limitSoc"`

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
	lp.SetPlanEnergy(payload.PlanTime, payload.PlanEnergy)
	lp.SetLimitEnergy(payload.LimitEnergy)
	lp.SetLimitSoc(payload.LimitSoc)

	// TODO mode warning
	lp.SetSocConfig(payload.Soc)

	mode, err := api.ChargeModeString(payload.DefaultMode)
	if err == nil {
		lp.SetDefaultMode(mode)
	}

	if err == nil {
		err = lp.SetPhasesConfigured(payload.PhasesConfigured)
	}

	if err == nil && payload.MinCurrent != 0 {
		err = lp.SetMinCurrent(payload.MinCurrent)
	}
	if err == nil && payload.MaxCurrent != 0 {
		err = lp.SetMaxCurrent(payload.MaxCurrent)
	}

	return err
}
