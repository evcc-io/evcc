package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/aiways"
	"github.com/evcc-io/evcc/vehicle/vw"
)

// https://github.com/davidgiga1993/AiwaysAPI
// https://github.com/TA2k/ioBroker.vw-connect

// Aiways is an api.Vehicle implementation for Aiways cars
type Aiways struct {
	*embed
	*vw.Provider // provides the api implementations
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

	_, err := api.Vehicles()

	// idk := idkproxy.New(log, Aiways.IDKParams)
	// ts, err := service.MbbTokenSource(log, idk, Aiways.AuthClientID, Aiways.AuthParams, cc.User, cc.Password)
	// if err != nil {
	// 	return nil, err
	// }

	// api := vw.NewAPI(log, ts, Aiways.Brand, Aiways.Country)
	// api.Client.Timeout = cc.Timeout

	// cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	// if err == nil {
	// 	if err = api.HomeRegion(cc.VIN); err == nil {
	// 		v.Provider = vw.NewProvider(api, cc.VIN, cc.Cache)
	// 	}
	// }

	return v, err
}
