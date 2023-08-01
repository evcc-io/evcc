package charger

import (
	"errors"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/homewizard"
	"github.com/evcc-io/evcc/util"
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
	cc := struct {
		embed        `mapstructure:",squash"`
		URI          string
		StandbyPower float64
		Cache        time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHomeWizard(cc.embed, cc.URI, cc.StandbyPower, cc.Cache)
}

// NewHomeWizard creates HomeWizard charger
func NewHomeWizard(embed embed, uri string, standbypower float64, cache time.Duration) (*HomeWizard, error) {
	conn, err := homewizard.NewConnection(uri, cache)
	if err != nil {
		return nil, err
	}

	c := &HomeWizard{
		conn: conn,
	}

	// Check compatible product type
	if c.conn.ProductType != "HWE-SKT" {
		return nil, errors.New("unsupported product type: " + c.conn.ProductType)
	}

	c.switchSocket = NewSwitchSocket(&embed, c.Enabled, c.conn.CurrentPower, standbypower)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *HomeWizard) Enabled() (bool, error) {
	return c.conn.Enabled()
}

// Enable implements the api.Charger interface
func (c *HomeWizard) Enable(enable bool) error {
	return c.conn.Enable(enable)
}

var _ api.MeterEnergy = (*HomeWizard)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *HomeWizard) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}
