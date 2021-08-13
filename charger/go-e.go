package charger

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/util/request"
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
	Fwv string `json:"fwv"`        // firmware version - indicates local response
	Car int    `json:"car,string"` // car status
	Alw int    `json:"alw,string"` // allow charging
	Amp int    `json:"amp,string"` // current [A]
	Err int    `json:"err,string"` // error
	Stp int    `json:"stp,string"` // stop state
	Tmp int    `json:"tmp,string"` // temperature [°C]
	Dws int    `json:"dws,string"` // energy [Ws]
	Nrg []int  `json:"nrg"`        // voltage, current, power
	Uby int    `json:"uby,string"` // unlocked_by
	Rna string `json:"rna"`        // RFID 1
	Rnm string `json:"rnm"`        // RFID 2
	Rne string `json:"rne"`        // RFID 3
	Rn4 string `json:"rn4"`        // RFID 4
	Rn5 string `json:"rn5"`        // RFID 5
	Rn6 string `json:"rn6"`        // RFID 6
	Rn7 string `json:"rn7"`        // RFID 7
	Rn8 string `json:"rn8"`        // RFID 8
	Rn9 string `json:"rn9"`        // RFID 9
	Rn1 string `json:"rn1"`        // RFID 10
}

func (g goeStatusResponse) RFIDName() string {
	switch g.Uby {
	case 1:
		return g.Rna
	case 2:
		return g.Rnm
	case 3:
		return g.Rne
	case 4:
		return g.Rn4
	case 5:
		return g.Rn5
	case 6:
		return g.Rn6
	case 7:
		return g.Rn7
	case 8:
		return g.Rn8
	case 9:
		return g.Rn9
	case 10:
		return g.Rn1
	default:
		return ""
	}
}

// GoE charger implementation
type GoE struct {
	*request.Helper
	uri, token string
	cache      time.Duration
	updated    time.Time
	status     goeStatusResponse
}

func init() {
	registry.Add("go-e", NewGoEFromConfig)
}

// NewGoEFromConfig creates a go-e charger from generic config
func NewGoEFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
		Token string
		URI   string
		Cache time.Duration
	}{}
	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.URI != "" && cc.Token != "" {
		return nil, errors.New("go-e config: should only have one of uri/token")
	}
	if cc.URI == "" && cc.Token == "" {
		return nil, errors.New("go-e config: must have one of uri/token")
	}

	return NewGoE(cc.URI, cc.Token, cc.Cache)
}

// NewGoE creates GoE charger
func NewGoE(uri, token string, cache time.Duration) (*GoE, error) {
	c := &GoE{
		Helper: request.NewHelper(util.NewLogger("go-e")),
		uri:    strings.TrimRight(uri, "/"),
		token:  strings.TrimSpace(token),
	}

	return c, nil
}

func (c *GoE) localResponse(function, payload string) (goeStatusResponse, error) {
	var status goeStatusResponse

	url := fmt.Sprintf("%s/%s", c.uri, function)
	if payload != "" {
		url += "?payload=" + payload
	}

	err := c.GetJSON(url, &status)
	return status, err
}

func (c *GoE) cloudResponse(function, payload string) (goeStatusResponse, error) {
	var status goeCloudResponse

	url := fmt.Sprintf("%s/%s?token=%s", goeCloud, function, c.token)
	if payload != "" {
		url += "&payload=" + payload
	}

	err := c.GetJSON(url, &status)
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

// apiUpdate invokes either cloud or local api
// goeStatusResponse is only valid for local api. Use Fwv if valid.
func (c *GoE) apiUpdate(payload string) (goeStatusResponse, error) {
	if c.token == "" {
		// let charger settle after update
		defer time.Sleep(2 * time.Second)
		return c.localResponse("mqtt", payload)
	}

	status, err := c.cloudResponse("api", payload)
	if err == nil {
		c.updated = time.Now()
		c.status = status
	}

	return status, err
}

// Status implements the api.Charger interface
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

// Enabled implements the api.Charger interface
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

// Enable implements the api.Charger interface
func (c *GoE) Enable(enable bool) error {
	var b int
	if enable {
		b = 1
	}

	status, err := c.apiUpdate(fmt.Sprintf("alw=%d", b))
	if err == nil {
		if status, err = c.apiStatus(); err == nil && status.Alw != b {
			return fmt.Errorf("alw update failed: %d", status.Alw)
		}
	}

	return err
}

// MaxCurrent implements the api.Charger interface
func (c *GoE) MaxCurrent(current int64) error {
	status, err := c.apiUpdate(fmt.Sprintf("amx=%d", current))
	if err == nil {
		if status, err = c.apiStatus(); err == nil && int64(status.Amp) != current {
			return fmt.Errorf("amp update failed: %d", status.Amp)
		}
	}

	return err
}

var _ api.Meter = (*GoE)(nil)

// CurrentPower implements the api.Meter interface
func (c *GoE) CurrentPower() (float64, error) {
	status, err := c.apiStatus()
	var power float64
	if len(status.Nrg) == 16 {
		power = float64(status.Nrg[11]) * 10
	}
	return power, err
}

var _ api.ChargeRater = (*GoE)(nil)

// ChargedEnergy implements the api.ChargeRater interface
func (c *GoE) ChargedEnergy() (float64, error) {
	status, err := c.apiStatus()
	energy := float64(status.Dws) / 3.6e5 // Deka-Watt-Seconds to kWh (100.000 == 0,277kWh)
	return energy, err
}

var _ api.MeterCurrent = (*GoE)(nil)

// Currents implements the api.MeterCurrent interface
func (c *GoE) Currents() (float64, float64, float64, error) {
	status, err := c.apiStatus()
	if len(status.Nrg) == 16 {
		return float64(status.Nrg[4]) / 10, float64(status.Nrg[5]) / 10, float64(status.Nrg[6]) / 10, nil
	}
	return 0, 0, 0, err
}

var _ api.Identifier = (*GoE)(nil)

// Identify implements the api.Identifier interface
func (c *GoE) Identify() (string, error) {
	status, err := c.apiStatus()
	return status.RFIDName(), err
}
