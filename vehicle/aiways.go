package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/aiways"
)

// https://github.com/davidgiga1993/AiwaysAPI
// https://github.com/TA2k/ioBroker.vw-connect

// Aiways is an api.Vehicle implementation for Aiways cars
type Aiways struct {
	*embed
	*aiways.Provider // provides the api implementations
}

func init() {
	registry.Add("aiways", NewAiwaysFromConfig)
}

// NewAiwaysFromConfig creates a new vehicle
func NewAiwaysFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
		Timeout             time.Duration
	}{
		Cache:   interval,
		Timeout: request.Timeout,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &Aiways{
		embed: &cc.embed,
	}

	log := util.NewLogger("aiways").Redact(cc.User, cc.Password, cc.VIN)

	api := aiways.NewAPI(log, cc.User, cc.Password)
	api.Client.Timeout = cc.Timeout

	_, err := api.Vehicles()

	// cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		v.Provider = aiways.NewProvider(api, cc.VIN, cc.Cache)
	}

	return v, err
}
