package meter

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/homewizard"
	"github.com/evcc-io/evcc/util"
)

// HomeWizard meter implementation
type HomeWizard struct {
	conn *homewizard.Connection
}

// HomeWizard meter implementation
func init() {
	registry.Add("homewizard", NewHomeWizardFromConfig)
}

// NewHomeWizardFromConfig creates a HomeWizard meter from generic config
func NewHomeWizardFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI   string
		Usage string
		Cache time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return NewHomeWizard(cc.URI, cc.Usage, cc.Cache)
}

// NewHomeWizard creates HomeWizard meter
func NewHomeWizard(uri string, usage string, cache time.Duration) (*HomeWizard, error) {
	conn, err := homewizard.NewConnection(uri, usage, cache)
	if err != nil {
		return nil, err
	}

	c := &HomeWizard{
		conn: conn,
	}

	return c, nil
}

var _ api.Meter = (*HomeWizard)(nil)

// CurrentPower implements the api.Meter interface
func (c *HomeWizard) CurrentPower() (float64, error) {
	return c.conn.CurrentPower()
}

var _ api.MeterEnergy = (*HomeWizard)(nil)

// TotalEnergy implements the api.MeterEnergy interface
func (c *HomeWizard) TotalEnergy() (float64, error) {
	return c.conn.TotalEnergy()
}

var _ api.PhaseCurrents = (*HomeWizard)(nil)

// Currents implements the api.PhaseCurrents interface
func (c *HomeWizard) Currents() (float64, float64, float64, error) {
	return c.conn.Currents()
}

var _ api.PhaseVoltages = (*HomeWizard)(nil)

// Voltages implements the api.PhaseVoltages interface
func (c *HomeWizard) Voltages() (float64, float64, float64, error) {
	return c.conn.Voltages()
}
