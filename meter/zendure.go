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

// NewZendureFromConfig creates a Zendure meter from generic config
func NewZendureFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
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

	return c, err
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
		return float64(res.GridInputPower) - float64(res.OutputHomePower), nil
	default:
		return 0, fmt.Errorf("invalid usage: %s", c.usage)
	}
}

// Soc implements the api.Battery interface
func (c *Zendure) Soc() (float64, error) {
	res, err := c.conn.Data()
	if err != nil {
		return 0, err
	}
	return float64(res.ElectricLevel), nil
}
