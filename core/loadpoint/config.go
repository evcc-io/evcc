package loadpoint

import (
	"fmt"
	"math"
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
	Title                    string             `json:"title"`
	DefaultMode              string             `json:"defaultMode"`
	AlwaysCharge             string             `json:"alwaysCharge"`
	Priority                 int                `json:"priority"`
	PhasesConfigured         int                `json:"phasesConfigured"`
	MinCurrent               float64            `json:"minCurrent"`
	MaxCurrent               float64            `json:"maxCurrent"`
	SmartCostLimit           *float64           `json:"smartCostLimit"`
	SmartFeedInPriorityLimit *float64           `json:"smartFeedInPriorityLimit"`
	SolarShare               float64            `json:"solarShare"`
	PlanEnergy               float64            `json:"planEnergy"`
	PlanTime                 time.Time          `json:"planTime"`
	PlanPrecondition_        int64              `json:"planPrecondition" mapstructure:"planPrecondition"` // TODO deprecated, keep for compatibility
	BatteryBoostLimit        int                `json:"batteryBoostLimit"`
	LimitEnergy              float64            `json:"limitEnergy"`
	LimitSoc                 int                `json:"limitSoc"`
	MinSoc                   int                `json:"minSoc"`
	Dehumidifier             DehumidifierConfig `json:"dehumidifier"`

	PlanStrategy api.PlanStrategy `json:"planStrategy"`

	Thresholds ThresholdsConfig `json:"thresholds"`
	Soc        SocConfig        `json:"soc"`
	UI         UIConfig         `json:"ui"`
}

// DehumidifierConfig configures humidity-based loadpoint control.
type DehumidifierConfig struct {
	TargetHumidity float64       `json:"targetHumidity"`
	Hysteresis     float64       `json:"hysteresis"`
	MinOnTime      time.Duration `json:"minOnTime"`
	MinOffTime     time.Duration `json:"minOffTime"`
}

// DefaultDehumidifierConfig returns the initial humidity control settings.
func DefaultDehumidifierConfig() DehumidifierConfig {
	return DehumidifierConfig{
		TargetHumidity: 50,
		Hysteresis:     2,
		MinOnTime:      10 * time.Minute,
		MinOffTime:     5 * time.Minute,
	}
}

func (c DehumidifierConfig) Validate() error {
	if math.IsNaN(c.TargetHumidity) || math.IsInf(c.TargetHumidity, 0) || c.TargetHumidity < 0 || c.TargetHumidity > 100 {
		return fmt.Errorf("target humidity must be between 0 and 100 %%RH")
	}
	if math.IsNaN(c.Hysteresis) || math.IsInf(c.Hysteresis, 0) || c.Hysteresis < 0 || c.Hysteresis > 100 {
		return fmt.Errorf("humidity hysteresis must be between 0 and 100 percentage points")
	}
	if c.MinOnTime < 0 || c.MinOffTime < 0 {
		return fmt.Errorf("minimum on/off times must not be negative")
	}
	return nil
}

// UIConfig holds display-only settings. Not used in control logic.
type UIConfig struct {
	MinTemp float64 `json:"minTemp"`
	MaxTemp float64 `json:"maxTemp"`
}

func SplitConfig(payload map[string]any) (DynamicConfig, map[string]any, error) {
	// split static and dynamic config via mapstructure
	var cc struct {
		DynamicConfig `mapstructure:",squash"`
		Other         map[string]any `mapstructure:",remain"`
	}
	cc.BatteryBoostLimit = 100 // default: disabled
	cc.SolarShare = 1          // default: full surplus
	cc.Dehumidifier = DefaultDehumidifierConfig()

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
	lp.SetSmartFeedInPriorityLimit(payload.SmartFeedInPriorityLimit)
	lp.SetSolarShare(payload.SolarShare)
	lp.SetThresholds(payload.Thresholds)
	lp.SetPlanEnergy(payload.PlanTime, payload.PlanEnergy)
	lp.SetPlanStrategy(payload.PlanStrategy)
	lp.SetBatteryBoostLimit(payload.BatteryBoostLimit)
	lp.SetLimitEnergy(payload.LimitEnergy)
	lp.SetLimitSoc(payload.LimitSoc)
	lp.SetMinSoc(payload.MinSoc)
	if err := lp.SetDehumidifierConfig(payload.Dehumidifier); err != nil {
		return err
	}

	// TODO mode warning
	lp.SetSocConfig(payload.Soc)
	lp.SetUI(payload.UI)

	mode, err := api.ChargeModeString(payload.DefaultMode)
	if err == nil {
		lp.SetDefaultMode(mode)
	}

	// always charge is optional; ignore "not supported" for devices without current control
	if payload.AlwaysCharge != "" {
		if ac, err := api.AlwaysChargeString(payload.AlwaysCharge); err == nil {
			_ = lp.SetAlwaysCharge(ac)
		}
	}

	if err == nil {
		err = lp.SetPhasesConfigured(payload.PhasesConfigured)
	}

	if err == nil {
		// In case both min and max current are set, we need to set them in the correct order to avoid validation errors
		switch {
		case payload.MinCurrent != 0 && payload.MaxCurrent != 0:
			if payload.MaxCurrent > lp.GetMaxCurrent() {
				if err = lp.SetMaxCurrent(payload.MaxCurrent); err == nil {
					err = lp.SetMinCurrent(payload.MinCurrent)
				}
			} else {
				if err = lp.SetMinCurrent(payload.MinCurrent); err == nil {
					err = lp.SetMaxCurrent(payload.MaxCurrent)
				}
			}
		case payload.MinCurrent != 0:
			err = lp.SetMinCurrent(payload.MinCurrent)
		case payload.MaxCurrent != 0:
			err = lp.SetMaxCurrent(payload.MaxCurrent)
		}
	}

	return err
}
