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
	usage             string
	conn              *zendure.Connection
	maxChargePower    int
	maxDischargePower int
}

// NewZendureFromConfig creates a Zendure meter from generic config
func NewZendureFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		batteryCapacity                `mapstructure:",squash"`
		batteryPowerLimits             `mapstructure:",squash"`
		batterySocLimits               `mapstructure:",squash"`
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

	m := &Zendure{
		usage:             cc.Usage,
		conn:              conn,
		maxChargePower:    int(cc.MaxChargePower),
		maxDischargePower: int(cc.MaxDischargePower),
	}

	// decorate battery
	if cc.Usage == "battery" {
		var setBatteryMode func(api.BatteryMode) error
		if m.maxDischargePower > 0 {
			setBatteryMode = m.batteryMode()
		}

		return decorateMeterBattery(
			m, nil, m.soc, cc.batteryCapacity.Decorator(),
			cc.batterySocLimits.Decorator(), cc.batteryPowerLimits.Decorator(), setBatteryMode,
		), nil
	}

	return m, nil
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

// batteryMode implements the api.BatteryController interface
func (c *Zendure) batteryMode() func(api.BatteryMode) error {
	return func(mode api.BatteryMode) error {
		switch mode {
		case api.BatteryNormal:
			props := map[string]int{
				"outputLimit": c.maxDischargePower,
			}
			if c.maxChargePower > 0 {
				props["acMode"] = 2 // output (discharge to home)
			}
			return c.conn.SetProperties(props)

		case api.BatteryHold:
			return c.conn.SetProperties(map[string]int{
				"outputLimit": 0,
			})

		case api.BatteryCharge:
			props := map[string]int{
				"outputLimit": 0,
			}
			if c.maxChargePower > 0 {
				props["acMode"] = 1 // input (charge from grid)
				props["inputLimit"] = c.maxChargePower
			}
			return c.conn.SetProperties(props)

		default:
			return api.ErrNotAvailable
		}
	}
}
