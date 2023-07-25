package charger

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/homewizard"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
)

// HomeWizard project homepage
// https://homewizard-energy-api.readthedocs.io/index.html

// HomeWizard charger implementation
type HomeWizard struct {
	conn *homewizard.Connection
	*switchSocket
}

func init() {
	registry.Add("homewizard", NewHomeWizardFromConfig)
}

// NewHomeWizardFromConfig creates a HomeWizard charger from generic config
func NewHomeWizardFromConfig(other map[string]interface{}) (api.Charger, error) {
	var cc = struct {
		embed        `mapstructure:",squash"`
		URI          string
		StandbyPower float64
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHomeWizard(cc.embed, cc.URI, cc.StandbyPower)
}

// NewHomeWizard creates HomeWizard charger
func NewHomeWizard(embed embed, uri string, standbypower float64) (*HomeWizard, error) {
	conn, err := homewizard.NewConnection(uri)
	if err != nil {
		return nil, err
	}

	c := &HomeWizard{
		conn: conn,
	}

	// Check compatible product type
	if c.conn.ProductType != "HWE-SKT" {
		return nil, errors.New("not supported product type: " + c.conn.ProductType)
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.conn.CurrentPower, standbypower)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *HomeWizard) Enabled() (bool, error) {
	var res homewizard.StateResponse
	err := c.conn.GetJSON(fmt.Sprintf("%s/data", c.conn.URI), &res)
	return res.PowerOn, err
}

// Enable implements the api.Charger interface
func (c *HomeWizard) Enable(enable bool) error {
	var res homewizard.StateResponse
	data := map[string]interface{}{
		"power_on": enable,
	}

	req, err := request.New(http.MethodPut, fmt.Sprintf("%s/state", c.conn.URI), request.MarshalJSON(data), request.JSONEncoding)
	if err != nil {
		return err
	}
	if err := c.conn.DoJSON(req, &res); err != nil {
		return err
	}

	switch {
	case enable && !res.PowerOn:
		return errors.New("switchOn failed")
	case !enable && res.PowerOn:
		return errors.New("switchOff failed")
	default:
		return nil
	}
}

var _ api.MeterEnergy = (*HomeWizard)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *HomeWizard) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
