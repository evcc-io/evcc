package meter

import (
	"fmt"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/zendure"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("zendure", NewZendureFromConfig)
}

type Zendure struct {
	usage string
	conn  *zendure.Connection
}

//go:generate go tool decorate -f decorateZendure -b *Zendure -r api.Meter -t "api.Battery,Soc,func() (float64, error)" -t "api.BatteryCapacity,Capacity,func() float64"

// NewZendureFromConfig creates a Zendure meter from generic config
func NewZendureFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		batteryCapacity                `mapstructure:",squash"`
		Usage, Account, Serial, Region string
		Timeout                        time.Duration
	}{
		Region:  "EU",
		Timeout: 30 * time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	conn, err := zendure.NewConnection(strings.ToUpper(cc.Region), cc.Account, cc.Serial, cc.Timeout)
	if err != nil {
		return nil, err
	}

	c := &Zendure{
		usage: cc.Usage,
		conn:  conn,
	}

	// decorate battery
	var soc func() (float64, error)
	if cc.Usage == "battery" {
		soc = c.soc
	}

	return decorateZendure(c, soc, cc.batteryCapacity.Decorator()), nil
}

// CurrentPower implements the api.Meter interface
func (c *Zendure) CurrentPower() (float64, error) {
	res, err := c.conn.Data()
	if err != nil {
		return 0, err
	}

	switch c.usage {
	case "pv":
		return float64(res.SolarInputPower), nil
	case "battery":
		return float64(res.PackInputPower) - float64(res.OutputPackPower), nil
	default:
		return 0, fmt.Errorf("invalid usage: %s", c.usage)
	}
}

// soc implements the api.Battery interface
func (c *Zendure) soc() (float64, error) {
	res, err := c.conn.Data()
	if err != nil {
		return 0, err
	}
	return float64(res.ElectricLevel), nil
}
