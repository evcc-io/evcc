package charger

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/andig/evcc/api"
)

const (
	goeStatus apiFunction = "status"
)

// goeStatusResponse is the API response if status not OK
type goeStatusResponse struct {
	Car uint   `yaml:"car"` // car status
	Alw uint   `yaml:"alw"` // allow charging
	Amp uint   `yaml:"amp"` // current [A]
	Err uint   `yaml:"err"`
	Stp uint   `yaml:"stp"`
	Tmp uint   `yaml:"tmp"`
	Dws uint   `yaml:"dws"` // energy [Ws]
	Nrg []uint `yaml:"nrg"` // voltage, current, power
}

// GoE charger implementation
type GoE struct {
	*api.HTTPHelper
	URI string
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
		URI:        URI,
	}

	c.HTTPHelper.Log.WARN.Println("-- experimental --")

	return c
}

func (c *GoE) apiURL(api apiFunction) string {
	return fmt.Sprintf("%s/%s", strings.TrimRight(c.URI, "/"), api)
}

func (c *GoE) getJSON(url string, result interface{}) error {
	b, err := c.GetJSON(url, result)
	if err != nil && len(b) > 0 {
		var error goeStatusResponse
		if err := json.Unmarshal(b, &error); err != nil {
			return err
		}

		return fmt.Errorf("response code: %d", error.Err)
	}

	return err
}

// Status implements the Charger.Status interface
func (c *GoE) Status() (api.ChargeStatus, error) {
	var status goeStatusResponse
	if err := c.getJSON(c.apiURL(goeStatus), status); err != nil {
		return api.StatusNone, err
	}

	switch status.Car {
	case 1:
		return api.StatusA, nil
	case 2:
		return api.StatusC, nil
	case 3:
		return api.StatusB, nil
	case 4:
		return api.StatusB, nil
	default:
		return api.StatusNone, fmt.Errorf("unknown result %d", status.Car)
	}
}

// Enabled implements the Charger.Enabled interface
func (c *GoE) Enabled() (bool, error) {
	var status goeStatusResponse
	if err := c.getJSON(c.apiURL(goeStatus), status); err != nil {
		return false, err
	}

	switch status.Alw {
	case 0:
		return false, nil
	case 1:
		return true, nil
	default:
		return false, fmt.Errorf("unknown result %d", status.Alw)
	}
}

// Enable implements the Charger.Enable interface
func (c *GoE) Enable(enable bool) error {
	var status goeStatusResponse

	uri := c.apiURL(goeStatus) + "/mqtt?alw="
	if enable {
		uri += "1"
	} else {
		uri += "0"
	}

	return c.getJSON(uri, status)
}

// MaxCurrent implements the Charger.MaxCurrent interface
func (c *GoE) MaxCurrent(current int64) error {
	var status goeStatusResponse
	uri := fmt.Sprintf(c.apiURL(goeStatus)+"/mqtt?amp=%d", current)
	return c.getJSON(uri, status)
}

// CurrentPower implements the Meter interface.
func (c *GoE) CurrentPower() (float64, error) {
	var status goeStatusResponse
	err := c.getJSON(c.apiURL(goeStatus), status)
	power := float64(status.Nrg[11]) * 10
	return power, err
}

// ChargedEnergy implements the ChargeRater interface.
func (c *GoE) ChargedEnergy() (float64, error) {
	var status goeStatusResponse
	err := c.getJSON(c.apiURL(goeStatus), status)
	energy := float64(status.Dws) / 3.6e6
	return energy, err
}
