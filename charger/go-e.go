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
	log *api.Logger
	URI string
}

// NewGoEFromConfig creates a go-e charger from generic config
func NewGoEFromConfig(log *api.Logger, other map[string]interface{}) api.Charger {
	cc := struct{ URI string }{}
	decodeOther(log, other, &cc)

	return NewGoE(cc.URI)
}

// NewGoE creates GoE charger
func NewGoE(URI string) *GoE {
	c := &GoE{
		URI: URI,
		log: api.NewLogger("go-e"),
	}

	c.log.WARN.Println("-- experimental --")

	return c
}

func (c *GoE) apiURL(api apiFunction) string {
	return fmt.Sprintf("%s/%s", strings.TrimRight(c.URI, "/"), api)
}

func (c *GoE) getJSON(url string, result interface{}) error {
	resp, body, err := getJSON(url, result)
	c.log.TRACE.Printf("GET %s: %s", url, string(body))

	if err != nil && len(body) == 0 {
		return err
	}

	var error goeStatusResponse
	_ = json.Unmarshal(body, &error)

	return fmt.Errorf("api %d: %d", resp.StatusCode, error.Err)
}

// Status implements the Charger.Status interface
func (c *GoE) Status() (api.ChargeStatus, error) {
	var status goeStatusResponse
	err := c.getJSON(c.apiURL(goeStatus), status)

	switch status.Car {
	case 1:
		return api.StatusA, err
	case 2:
		return api.StatusC, err
	case 3:
		return api.StatusB, err
	case 4:
		return api.StatusB, err
	}

	return api.StatusNone, fmt.Errorf("unknown result %d", status.Car)
}

// Enabled implements the Charger.Enabled interface
func (c *GoE) Enabled() (bool, error) {
	var status goeStatusResponse
	err := c.getJSON(c.apiURL(goeStatus), status)

	switch status.Alw {
	case 0:
		return false, err
	case 1:
		return true, err
	}

	return false, fmt.Errorf("unknown result %d", status.Alw)
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
