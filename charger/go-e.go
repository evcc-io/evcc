package charger

import (
	"fmt"
	"strings"

	"github.com/andig/evcc/api"
)

const (
	goeStatus  apiFunction = "status"
	goePayload apiFunction = "mqtt?payload="
)

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
	*api.HTTPHelper
	uri string
}

// NewGoEFromConfig creates a go-e charger from generic config
func NewGoEFromConfig(log *api.Logger, other map[string]interface{}) api.Charger {
	cc := struct{ URI string }{}
	api.DecodeOther(log, other, &cc)

	return NewGoE(cc.URI)
}

// NewGoE creates GoE charger
func NewGoE(URI string) *GoE {
	c := &GoE{
		HTTPHelper: api.NewHTTPHelper(api.NewLogger("go-e")),
		uri:        strings.TrimRight(URI, "/"),
	}

	return c
}

func (c *GoE) apiURL(api apiFunction) string {
	return fmt.Sprintf("%s/%s", c.uri, api)
}

// Status implements the Charger.Status interface
func (c *GoE) Status() (api.ChargeStatus, error) {
	var status goeStatusResponse
	if _, err := c.GetJSON(c.apiURL(goeStatus), &status); err != nil {
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
	var status goeStatusResponse
	if _, err := c.GetJSON(c.apiURL(goeStatus), &status); err != nil {
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
	var status goeStatusResponse

	var b int
	if enable {
		b = 1
	}

	uri := c.apiURL(goePayload) + fmt.Sprintf("alw=%d", b)

	_, err := c.GetJSON(uri, &status)
	if err == nil && status.Alw != b {
		return fmt.Errorf("alw update failed: %d", status.Amp)
	}

	return err
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (c *GoE) MaxCurrent(current int64) error {
	var status goeStatusResponse
	uri := c.apiURL(goePayload) + fmt.Sprintf("amp=%d", current)

	_, err := c.GetJSON(uri, &status)
	if err == nil && int64(status.Amp) != current {
		return fmt.Errorf("amp update failed: %d", status.Amp)
	}

	return err
}

// CurrentPower implements the Meter interface.
func (c *GoE) CurrentPower() (float64, error) {
	var status goeStatusResponse
	_, err := c.GetJSON(c.apiURL(goeStatus), &status)
	var power float64
	if len(status.Nrg) == 16 {
		power = float64(status.Nrg[11]) * 10
	}
	return power, err
}

// ChargedEnergy implements the ChargeRater interface
func (c *GoE) ChargedEnergy() (float64, error) {
	var status goeStatusResponse
	_, err := c.GetJSON(c.apiURL(goeStatus), &status)
	energy := float64(status.Dws) / 3.6e6
	return energy, err
}
