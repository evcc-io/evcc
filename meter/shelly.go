package meter

import (
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
	var cc struct {
		URI      string
		User     string
		Password string
		Channel  int
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return shelly.NewConnection(cc.URI, cc.User, cc.Password, cc.Channel)
}
