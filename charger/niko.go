package charger

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/niko"
	"github.com/evcc-io/evcc/util"
)

// Niko Home Control smart plug via Hobby API

func init() {
	registry.Add("niko", NewNikoFromConfig)
}

// Niko charger implementation
type Niko struct {
	*switchSocket
	conn *niko.Connection
}

// NewNikoFromConfig creates a Niko Home Control smart plug charger from generic config
func NewNikoFromConfig(other map[string]any) (api.Charger, error) {
	cc := struct {
		embed        `mapstructure:",squash"`
		Host         string
		User         string
		Token        string
		Device       string
		StandbyPower float64
	}{
		User: "hobby",
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	conn, err := niko.NewConnection(cc.Host, cc.User, cc.Token, cc.Device)
	if err != nil {
		return nil, err
	}

	c := &Niko{conn: conn}
	c.switchSocket = NewSwitchSocket(&cc.embed, c.Enabled, conn.CurrentPower, cc.StandbyPower)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Niko) Enabled() (bool, error) {
	return c.conn.Enabled()
}

// Enable implements the api.Charger interface
func (c *Niko) Enable(enable bool) error {
	return c.conn.Enable(enable)
}
