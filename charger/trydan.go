package charger

// https://v2charge.com/trydan/

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/util/sponsor"
)

type RealTimeData struct {
	ID               string  `json:"ID"`
	ChargeState      int     `json:"ChargeState"`
	ReadyState       int     `json:"ReadyState"`
	ChargePower      float64 `json:"ChargePower"`
	ChargeEnergy     float64 `json:"ChargeEnergy"`
	SlaveError       int     `json:"SlaveError"`
	ChargeTime       int     `json:"ChargeTime"`
	HousePower       float64 `json:"HousePower"`
	FVPower          float64 `json:"FVPower"`
	BatteryPower     float64 `json:"BatteryPower"`
	Paused           int     `json:"Paused"`
	Locked           int     `json:"Locked"`
	Timer            int     `json:"Timer"`
	Intensity        int     `json:"Intensity"`
	Dynamic          int     `json:"Dynamic"`
	MinIntensity     int     `json:"MinIntensity"`
	MaxIntensity     int     `json:"MaxIntensity"`
	PauseDynamic     int     `json:"PauseDynamic"`
	FirmwareVersion  string  `json:"FirmwareVersion"`
	DynamicPowerMode int     `json:"DynamicPowerMode"`
	ContractedPower  int     `json:"ContractedPower"`
}

// Trydan charger implementation
type Trydan struct {
	*request.Helper
	uri     string
	statusG util.Cacheable[RealTimeData]
	current int
	enabled bool
}

func init() {
	registry.Add("trydan", NewTrydanFromConfig)
}

// NewTrydanFromConfig creates a Trydan charger from generic config
func NewTrydanFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		URI   string
		Cache time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI == "" {
		return nil, errors.New("missing uri")
	}

	return NewTrydan(cc.URI, cc.Cache)
}

// NewTrydan creates Trydan charger
func NewTrydan(uri string, cache time.Duration) (api.Charger, error) {
	if !sponsor.IsAuthorized() {
		return nil, api.ErrSponsorRequired
	}

	c := &Trydan{
		Helper: request.NewHelper(util.NewLogger("trydan")),
		uri:    util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
	}

	c.statusG = util.ResettableCached(func() (RealTimeData, error) {
		var res RealTimeData
		uri := fmt.Sprintf("%s/RealTimeData", c.uri)
		err := c.GetJSON(uri, &res)
		return res, err
	}, cache)

	return c, nil
}

// Status implements the api.Charger interface
func (t Trydan) Status() (api.ChargeStatus, error) {
	data, err := t.statusG.Get()
	if err != nil {
		return api.StatusNone, err
	}
	switch state := data.ChargeState; state {
	case 0:
		return api.StatusA, nil
	case 1:
		return api.StatusB, nil
	case 2:
		return api.StatusC, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown status: %d", state)
	}
}

// Enabled implements the api.Charger interface
func (c Trydan) Enabled() (bool, error) {
	data, err := c.statusG.Get()
	return data.Paused == 0 && data.Locked == 0, err
}

func (c *Trydan) setValue(param string, value int) error {
	uri := fmt.Sprintf("%s/write/%s=%d", c.uri, param, value)
	res, err := c.GetBody(uri)
	if str := string(res); err == nil && str != "OK" {
		err = fmt.Errorf("command failed: %s", res)
	}
	return err
}

// Enable implements the api.Charger interface
func (c Trydan) Enable(enable bool) error {
	var pause int
	if !enable {
		pause = 1
	}

	if err := c.setValue("Paused", pause); err != nil {
		return err
	}
	if err := c.setValue("Locked", pause); err != nil {
		return err
	}
	c.enabled = enable

	return nil
}

// MaxCurrent implements the api.Charger interface
func (c Trydan) MaxCurrent(current int64) error {
	err := c.setValue("Intensity", int(current))
	if err == nil {
		c.current = int(current)
	}
	return err
}

var _ api.ChargeRater = (*Trydan)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c Trydan) ChargedEnergy() (float64, error) {
	data, err := c.statusG.Get()
	return data.ChargeEnergy, err
}

var _ api.ChargeTimer = (*Trydan)(nil)

// ChargeDuration implements the api.ChargeTimer interface
func (c Trydan) ChargeDuration() (time.Duration, error) {
	data, err := c.statusG.Get()
	return time.Duration(data.ChargeTime) * time.Second, err
}

var _ api.Meter = (*Trydan)(nil)

// CurrentPower implements the api.Meter interface
func (c Trydan) CurrentPower() (float64, error) {
	data, err := c.statusG.Get()
	return data.ChargePower, err
}

var _ api.Diagnosis = (*Trydan)(nil)

// Diagnose implements the api.Diagnosis interface
func (c *Trydan) Diagnose() {
	data, err := c.statusG.Get()
	if err != nil {
		fmt.Printf("%#v", data)
	}
}
