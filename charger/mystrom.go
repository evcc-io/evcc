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
	conn *mystrom.Connection
	*switchSocket
	cache   time.Duration
	reportG func() (mystrom.Report, error)
}

// NewMyStromFromConfig creates a myStrom charger from generic config
func NewMyStromFromConfig(other map[string]interface{}) (api.Charger, error) {
	cc := struct {
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
		conn:  mystrom.NewConnection(cc.URI),
		cache: cc.Cache,
	}

	c.switchSocket = NewSwitchSocket(c.Enabled, c.conn.CurrentPower, cc.StandbyPower)
	c.reportG = provider.Cached(c.conn.Report, c.cache)

	return c, nil
}

// Enabled implements the api.Charger interface
func (c *MyStrom) Enabled() (bool, error) {
	res, err := c.reportG()
	return res.Relay, err
}

// Enable implements the api.Charger interface
func (c *MyStrom) Enable(enable bool) error {
	// reset cache
	c.reportG = provider.Cached(c.conn.Report, c.cache)

	onoff := map[bool]int{false: 0, true: 1}
	return c.conn.Request(fmt.Sprintf("relay?state=%d", onoff[enable]))
}

// MaxCurrent implements the api.Charger interface
func (c *MyStrom) MaxCurrent(current int64) error {
	return nil
}
