package meter

import (
	"errors"
	"strings"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/meter/bosch"
	"github.com/evcc-io/evcc/util"
)

type BoschBpts5Hybrid struct {
	implement.Caps
	api   *bosch.API
	usage string
}

func init() {
	registry.Add("bosch-bpt", NewBoschBpts5HybridFromConfig)
}

// NewBoschBpts5HybridFromConfig creates a Bosch BPT-S 5 Hybrid Meter from generic config
func NewBoschBpts5HybridFromConfig(other map[string]any) (api.Meter, error) {
	var cc struct {
		batteryCapacity    `mapstructure:",squash"`
		batteryPowerLimits `mapstructure:",squash"`
		batterySocLimits   `mapstructure:",squash"`
		URI                string
		Usage              string
		Cache              time.Duration
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Usage == "" {
		return nil, errors.New("missing usage")
	}

	return NewBoschBpts5Hybrid(cc.URI, cc.Usage, cc.Cache, cc.batteryCapacity.Decorator(), cc.batterySocLimits.Decorator(), cc.batteryPowerLimits.Decorator())
}

// NewBoschBpts5Hybrid creates a Bosch BPT-S 5 Hybrid Meter
func NewBoschBpts5Hybrid(uri, usage string, cache time.Duration, capacity func() float64, batterySocLimits, batteryPowerLimits func() (float64, float64)) (*BoschBpts5Hybrid, error) {
	log := util.NewLogger("bosch-bpt")

	instance, exists := bosch.Instances.LoadOrStore(uri, bosch.NewLocal(log, uri, cache))
	if !exists {
		if err := instance.(*bosch.API).Login(); err != nil {
			return nil, err
		}
	}

	m := &BoschBpts5Hybrid{
		Caps:  implement.New(),
		api:   instance.(*bosch.API),
		usage: strings.ToLower(usage),
	}

	if usage == "battery" {
		implement.Has(m, implement.Battery(m.soc))
		implement.May(m, implement.BatteryCapacity(capacity))
		implement.May(m, implement.BatterySocLimiter(batterySocLimits))
		implement.May(m, implement.BatteryPowerLimiter(batteryPowerLimits))
	}

	return m, nil
}

// CurrentPower implements the api.Meter interface
func (m *BoschBpts5Hybrid) CurrentPower() (float64, error) {
	status, err := m.api.Status()

	switch m.usage {
	case "grid":
		return status.BuyFromGrid - status.SellToGrid, err
	case "pv":
		return status.PvPower, err
	case "battery":
		return status.BatteryDischargePower - status.BatteryChargePower, err
	default:
		return 0, err
	}
}

// soc implements the api.Battery interface
func (m *BoschBpts5Hybrid) soc() (float64, error) {
	status, err := m.api.Status()
	return status.CurrentBatterySoc, err
}
