package charger

// https://v2charge.com/trydan/

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
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
	log *util.Logger
	*request.Helper
	uri     string
	statusG provider.Cacheable[RealTimeData]
	current int
	enabled bool
}

func init() {
	registry.Add("trydan", NewTrydanFromConfig)
}

// NewTrydanFromConfig creates a Trydan charger from generic config
func NewTrydanFromConfig(other map[string]interface{}) (api.Charger, error) {
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
	log := util.NewLogger("trydan")
	c := &Trydan{
		log:    log,
		Helper: request.NewHelper(log),
		uri:    util.DefaultScheme(strings.TrimSuffix(uri, "/"), "http"),
	}

	c.statusG = provider.ResettableCached(func() (RealTimeData, error) {
		var res RealTimeData
		uri := fmt.Sprintf("%s/RealTimeData", c.uri)
		err := c.GetJSON(uri, &res)
		log.DEBUG.Printf("Trydan status response %#v", res)

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
		return api.StatusF, nil
	}
}

// Enabled implements the api.Charger interface
func (c Trydan) Enabled() (bool, error) {
	data, err := c.statusG.Get()
	ret := data.Paused == 0 && data.Locked == 0
	c.log.DEBUG.Printf("Trydan Enabled: %t Paused: %d Locked: %d", ret, data.Paused, data.Locked)
	return ret, err
}

func setValue[T int | int64](c *Trydan, parameter string, value T) error {
	uri := fmt.Sprintf("%s/write/%s=%d", c.uri, parameter, value)
	c.log.DEBUG.Printf("Trydan Set URI: %s Value: %d", uri, value)
	res, err := c.GetBody(uri)
	if err == nil {
		resStr := string(res[:])
		if resStr != "OK" {
			err = fmt.Errorf("command failed: %s", res)
		}
	}
	return err
}

func (c *Trydan) setValueInt(parameter string, value int) error {
	return setValue(c, parameter, value)
}

func (c *Trydan) setValueInt64(parameter string, value int64) error {
	return setValue(c, parameter, value)
}

// Enable implements the api.Charger interface
func (c Trydan) Enable(enable bool) error {
	var _enable = 1

	if enable {
		_enable = 0
	}
	c.log.DEBUG.Printf("Trydan Set Paused: %d", _enable)
	err := c.setValueInt("Paused", _enable)
	if err != nil {
		return err
	}
	c.log.DEBUG.Printf("Trydan Set Locked: %d", _enable)
	err = c.setValueInt("Locked", _enable)
	if err != nil {
		return err
	}
	c.enabled = enable
	c.log.DEBUG.Printf("Trydan Enable: %t", c.enabled)
	return nil
}

// MaxCurrent implements the api.Charger interface
func (c Trydan) MaxCurrent(current int64) error {
	err := c.setValueInt64("Intensity", current)
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
