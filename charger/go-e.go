package charger

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
)

// https://go-e.co/app/api.pdf

const goeCloud = "https://api.go-e.co"

// goeCloudResponse is the cloud API response
type goeCloudResponse struct {
	Success *bool             `json:"success"` // only valid for cloud payload commands
	Age     int               `json:"age"`
	Error   string            `json:"error"` // only valid for cloud payload commands
	Data    goeStatusResponse `json:"data"`
}

// goeStatusResponse is the API response if status not OK
type goeStatusResponse struct {
	Car int   `json:"car,string"` // car status
	Alw int   `json:"alw,string"` // allow charging
	Amp int   `json:"amp,string"` // current [A]
	Err int   `json:"err,string"` // error
	Stp int   `json:"stp,string"` // stop state
	Tmp int   `json:"tmp,string"` // temperature [°C]
	Dws int   `json:"dws,string"` // energy [Ws]
	Nrg []int `json:"nrg"`        // voltage, current, power
}

// GoE charger implementation
type GoE struct {
	*util.HTTPHelper
	uri, token string
	cache      time.Duration
	updated    time.Time
	status     goeStatusResponse
}

// NewGoEFromConfig creates a go-e charger from generic config
func NewGoEFromConfig(log *util.Logger, other map[string]interface{}) api.Charger {
	cc := struct {
		Token string
		URI   string
		Cache time.Duration
	}{}
	util.DecodeOther(log, other, &cc)

	if cc.URI != "" && cc.Token != "" {
		log.FATAL.Fatal("config: should only have one of uri/token")
	}
	if cc.URI == "" && cc.Token == "" {
		log.FATAL.Fatal("config: must have one of uri/token")
	}

	return NewGoE(cc.URI, cc.Token, cc.Cache)
}

// NewGoE creates GoE charger
func NewGoE(uri, token string, cache time.Duration) *GoE {
	c := &GoE{
		HTTPHelper: util.NewHTTPHelper(util.NewLogger("go-e")),
		uri:        strings.TrimRight(uri, "/"),
		token:      strings.TrimSpace(token),
	}

	return c
}

func (c *GoE) localResponse(function, payload string) (goeStatusResponse, error) {
	var status goeStatusResponse

	url := fmt.Sprintf("%s/%s", c.uri, function)
	if payload != "" {
		url += "?payload=" + payload
	}

	_, err := c.GetJSON(url, &status)
	return status, err
}

func (c *GoE) cloudResponse(function, payload string) (goeStatusResponse, error) {
	var status goeCloudResponse

	url := fmt.Sprintf("%s/%s?token=%s", goeCloud, function, c.token)
	if payload != "" {
		url += "&payload=" + payload
	}

	_, err := c.GetJSON(url, &status)
	if err == nil && status.Success != nil && !*status.Success {
		err = errors.New(status.Error)
	}

	return status.Data, err
}

func (c *GoE) apiStatus() (status goeStatusResponse, err error) {
	if c.token == "" {
		return c.localResponse("status", "")
	}

	status = c.status // cached value

	if time.Since(c.updated) >= c.cache {
		status, err = c.cloudResponse("api_status", "")
		if err == nil {
			c.updated = time.Now()
			c.status = status
		}
	}

	return status, err
}

func (c *GoE) apiUpdate(payload string) (goeStatusResponse, error) {
	if c.token == "" {
		return c.localResponse("mqtt", payload)
	}

	status, err := c.cloudResponse("api", payload)
	if err == nil {
		c.updated = time.Now()
		c.status = status
	}

	return status, err
}

// Status implements the Charger.Status interface
func (c *GoE) Status() (api.ChargeStatus, error) {
	status, err := c.apiStatus()
	if err != nil {
		return api.StatusNone, err
	}

	switch status.Car {
	case 1:
		return api.StatusA, nil
	case 2:
		return api.StatusC, nil
	case 3, 4:
		return api.StatusB, nil
	default:
		return api.StatusNone, fmt.Errorf("car unknown result: %d", status.Car)
	}
}

// Enabled implements the Charger.Enabled interface
func (c *GoE) Enabled() (bool, error) {
	status, err := c.apiStatus()
	if err != nil {
		return false, err
	}

	switch status.Alw {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("alw unknown result: %d", status.Alw)
	}
}

// Enable implements the Charger.Enable interface
func (c *GoE) Enable(enable bool) error {
	var b int
	if enable {
		b = 1
	}

	status, err := c.apiUpdate(fmt.Sprintf("alw=%d", b))
	if err == nil && status.Alw != b {
		return fmt.Errorf("alw update failed: %d", status.Amp)
	}

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (c *GoE) MaxCurrent(current int64) error {
	status, err := c.apiUpdate(fmt.Sprintf("amp=%d", current))
	if err == nil && int64(status.Amp) != current {
		return fmt.Errorf("amp update failed: %d", status.Amp)
	}

	return err
}

// CurrentPower implements the Meter interface.
func (c *GoE) CurrentPower() (float64, error) {
	status, err := c.apiStatus()
	var power float64
	if len(status.Nrg) == 16 {
		power = float64(status.Nrg[11]) * 10
	}
	return power, err
}

// ChargedEnergy implements the ChargeRater interface
func (c *GoE) ChargedEnergy() (float64, error) {
	status, err := c.apiStatus()
	energy := float64(status.Dws) / 3.6e5 // Deka-Watt-Seconds to kWh (100.000 == 0,277kWh)
	return energy, err
}

// Currents implements the MeterCurrent interface
func (c *GoE) Currents() (float64, float64, float64, error) {
	status, err := c.apiStatus()
	if len(status.Nrg) == 16 {
		return float64(status.Nrg[4]) / 10, float64(status.Nrg[5]) / 10, float64(status.Nrg[6]) / 10, nil
	}
	return 0, 0, 0, err
}
