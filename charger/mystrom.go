package charger

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/mystrom"
	"github.com/evcc-io/evcc/provider"
	"github.com/evcc-io/evcc/util"
)

// myStrom switch:
// https://api.mystrom.ch/#fbb2c698-e37a-4584-9324-3f8b2f615fe2

func init() {
	registry.Add("mystrom", NewMyStromFromConfig)
}

// MyStrom charger implementation
type MyStrom struct {
	*switchSocket
	conn    *mystrom.Connection
	reportG provider.Cacheable[mystrom.Report]
}

// NewMyStromFromConfig creates a myStrom charger from generic config
func NewMyStromFromConfig(other map[string]interface{}) (api.Charger, error) {
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

	c := &MyStrom{
		conn: mystrom.NewConnection(cc.URI),
	}

	c.switchSocket = NewSwitchSocket(&cc.embed, c.Enabled, c.conn.CurrentPower, cc.StandbyPower)
	c.reportG = provider.ResettableCached(c.conn.Report, cc.Cache)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *MyStrom) Enabled() (bool, error) {
	res, err := c.reportG.Get()
	return res.Relay, err
}

// Enable implements the api.Charger interface
func (c *MyStrom) Enable(enable bool) error {
	c.reportG.Reset()

	onoff := map[bool]int{false: 0, true: 1}
	return c.conn.Request(fmt.Sprintf("relay?state=%d", onoff[enable]))
}
