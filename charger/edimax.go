package charger

import (
	"context"
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/edimax"
	"github.com/evcc-io/evcc/util"
)

// Edimax charger implementation
type Edimax struct {
	conn *edimax.Connection
	*switchSocket
}

func init() {
	registry.AddCtx("edimax", NewEdimaxFromConfig)
}

// NewEdimaxFromConfig creates an Edimax charger from generic config
func NewEdimaxFromConfig(ctx context.Context, other map[string]any) (api.Charger, error) {
	cc := struct {
		embed        `mapstructure:",squash"`
		Host         string
		User         string
		Password     string
		StandbyPower float64
		Cache        time.Duration
	}{
		User:     "admin",
		Password: "1234",
		Cache:    time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	c := &Edimax{
		conn: edimax.NewConnection(ctx, cc.Host, cc.User, cc.Password, cc.Cache),
	}

	c.switchSocket = NewSwitchSocket(&cc.embed, c.Enabled, c.conn.CurrentPower, cc.StandbyPower)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *Edimax) Enabled() (bool, error) {
	return c.conn.Enabled()
}

// Enable implements the api.Charger interface
func (c *Edimax) Enable(enable bool) error {
	if err := c.conn.Enable(enable); err != nil {
		return err
	}

	enabled, err := c.Enabled()
	switch {
	case err != nil:
		return err
	case enable != enabled:
		onoff := map[bool]string{true: "on", false: "off"}
		return fmt.Errorf("switch %s failed", onoff[enable])
	default:
		return nil
	}
}
