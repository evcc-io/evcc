package meter

import (
	"fmt"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/iotawatt"
	"github.com/evcc-io/evcc/util"
)

func init() {
	registry.Add("iotawatt", NewIoTaWattFromConfig)
}

// NewIoTaWattFromConfig creates an IoTaWatt meter from generic config
func NewIoTaWattFromConfig(other map[string]any) (api.Meter, error) {
	cc := struct {
		URI      string
		Channels []string
		Usage    string
	}{}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	// filter empty channel names (from unset template params)
	channels := make([]string, 0, len(cc.Channels))
	for _, ch := range cc.Channels {
		if ch != "" {
			channels = append(channels, ch)
		}
	}

	if len(channels) == 0 {
		return nil, fmt.Errorf("missing channels")
	}

	return NewIoTaWatt(cc.URI, channels)
}

// NewIoTaWatt creates an IoTaWatt meter
func NewIoTaWatt(uri string, channels []string) (api.Meter, error) {
	if len(channels) != 1 && len(channels) != 3 {
		return nil, fmt.Errorf("channels must have 1 (single-phase) or 3 (three-phase) entries, got %d", len(channels))
	}

	conn, err := iotawatt.NewConnection(uri)
	if err != nil {
		return nil, err
	}

	// validate all channels are Watts series
	if _, err := conn.ValidateSeries(channels, "Watts"); err != nil {
		return nil, err
	}

	// power getter — sum all channels
	powerG := func() (float64, error) {
		values, err := conn.QueryPower(channels...)
		if err != nil {
			return 0, err
		}
		var sum float64
		for _, v := range values {
			sum += v
		}
		return sum, nil
	}

	m, _ := NewConfigurable(powerG)

	// energy getter — sum Wh across all channels
	energyG := func() (float64, error) {
		return conn.TotalEnergy(channels...)
	}

	// per-phase getters (3-phase only)
	var currentsG, voltagesG, powersG func() (float64, float64, float64, error)

	if len(channels) == 3 {
		powersG = func() (float64, float64, float64, error) {
			values, err := conn.QueryPower(channels...)
			if err != nil {
				return 0, 0, 0, err
			}
			return values[0], values[1], values[2], nil
		}

		currentsG = func() (float64, float64, float64, error) {
			values, err := conn.QueryCurrents(channels...)
			if err != nil {
				return 0, 0, 0, err
			}
			return values[0], values[1], values[2], nil
		}

		voltagesG = func() (float64, float64, float64, error) {
			values, err := conn.QueryVoltages(channels...)
			if err != nil {
				return 0, 0, 0, err
			}
			return values[0], values[1], values[2], nil
		}
	}

	return m.Decorate(energyG, currentsG, voltagesG, powersG, nil), nil
}
