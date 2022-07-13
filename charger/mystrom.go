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
	*mystrom.Connection
	standbypower float64
	cache        time.Duration
	reportG      func() (mystrom.Report, error)
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
		Connection:   mystrom.NewConnection(cc.URI),
		standbypower: cc.StandbyPower,
		cache:        cc.Cache,
	}

	c.reportG = provider.Cached(c.Report, c.cache)

	return c, nil
}

var _ api.Meter = (*MyStrom)(nil)

// Status implements the api.Charger interface
func (c *MyStrom) Status() (api.ChargeStatus, error) {
	return switchStatus(c.Enabled, c.CurrentPower, c.standbypower)
}

// Enabled implements the api.Charger interface
func (c *MyStrom) Enabled() (bool, error) {
	res, err := c.reportG()
	return res.Relay, err
}

// Enable implements the api.Charger interface
func (c *MyStrom) Enable(enable bool) error {
	// reset cache
	c.reportG = provider.Cached(c.Report, c.cache)

	onoff := map[bool]int{false: 0, true: 1}
	return c.Request(fmt.Sprintf("relay/state=%d", onoff[enable]))
}

// MaxCurrent implements the api.Charger interface
func (c *MyStrom) MaxCurrent(current int64) error {
	return nil
}
