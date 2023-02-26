package meter

import (
	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/meter/shelly_pro_3em"
	"github.com/evcc-io/evcc/util"
)

// Shelly meter implementation
func init() {
	registry.Add("shelly-pro-3em", NewShellyPro3EmFromConfig)
}

// NewShellyFromConfig creates a Shelly energy meter from generic config
func NewShellyPro3EmFromConfig(other map[string]interface{}) (api.Meter, error) {
	var cc struct {
		URI      string
		User     string
		Password string
		Channel  int
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	return shelly_pro_3em.NewConnection(cc.URI, cc.User, cc.Password, cc.Channel)
}
