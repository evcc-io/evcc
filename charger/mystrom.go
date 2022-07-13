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

	wb := &MyStrom{
		Connection:   mystrom.NewConnection(cc.URI),
		standbypower: cc.StandbyPower,
		cache:        cc.Cache,
	}

	wb.reportG = provider.Cached(wb.Report, wb.cache)

	return wb, nil
}

var _ api.Meter = (*MyStrom)(nil)

// Status implements the api.Charger interface
func (wb *MyStrom) Status() (api.ChargeStatus, error) {
	return switchStatus(wb.Enabled, wb.CurrentPower, wb.standbypower)
}

// Enabled implements the api.Charger interface
func (wb *MyStrom) Enabled() (bool, error) {
	res, err := wb.reportG()
	return res.Relay, err
}

// Enable implements the api.Charger interface
func (wb *MyStrom) Enable(enable bool) error {
	// reset cache
	wb.reportG = provider.Cached(wb.Report, wb.cache)

	onoff := map[bool]int{false: 0, true: 1}
	return wb.Request(fmt.Sprintf("relay/state=%d", onoff[enable]))
}

// MaxCurrent implements the api.Charger interface
func (wb *MyStrom) MaxCurrent(current int64) error {
	return nil
}
