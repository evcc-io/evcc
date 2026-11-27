package tempsensor

import (
	"errors"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/implement"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/homeassistant"
)

func init() {
	registry.Add("homeassistant", NewHomeAssistantFromConfig)
}

// NewHomeAssistantFromConfig creates a Home Assistant temp sensor from generic config
func NewHomeAssistantFromConfig(other map[string]any) (api.Battery, error) {
	var cc struct {
		homeassistant.Config `mapstructure:",squash"`
		Temp                 string
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	if cc.Temp == "" {
		return nil, errors.New("missing temp entity")
	}

	conn, err := cc.Config.NewConnection(util.NewLogger("ha-tempsensor"))
	if err != nil {
		return nil, err
	}

	// GetTemperatureState converts °F/K readings to °C based on the entity's unit
	return &sensor{implement.Battery(func() (float64, error) {
		return conn.GetTemperatureState(cc.Temp)
	})}, nil
}
