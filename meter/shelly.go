package meter

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/shelly"
	"github.com/evcc-io/evcc/util"
)

// Shelly meter implementation
func init() {
	registry.Add("shelly", NewShellyFromConfig)
}

// NewShellyFromConfig creates a Shelly charger from generic config
func NewShellyFromConfig(other map[string]interface{}) (api.Meter, error) {
	cc := struct {
		URI      string
		User     string
		Password string
		Channel  int
		Cache    time.Duration
	}{
		Cache: time.Second,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	conn, err := shelly.NewConnection(cc.URI, cc.User, cc.Password, cc.Channel, cc.Cache)
	if err != nil {
		return nil, err
	}

	return shelly.NewSwitch(conn), nil
}
