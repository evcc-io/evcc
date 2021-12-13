package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logx"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/audi"
	"github.com/evcc-io/evcc/vehicle/vw"
)

// https://github.com/davidgiga1993/AudiAPI
// https://github.com/TA2k/ioBroker.vw-connect

// Audi is an api.Vehicle implementation for Audi cars
type Audi struct {
	*embed
	*vw.Provider // provides the api implementations
	// audiProvider *audi.Provider
}

func init() {
	registry.Add("audi", NewAudiFromConfig)
}

// NewAudiFromConfig creates a new vehicle
func NewAudiFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &Audi{
		embed: &cc.embed,
	}

	log := logx.Redact(logx.NewModule("audi"), cc.User, cc.Password, cc.VIN)

	identity := vw.NewIdentity(log, audi.AuthClientID, audi.AuthParams, cc.User, cc.Password)

	err := identity.Login()
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := vw.NewAPI(log, identity, "Audi", "DE")
	api.Client.Timeout = cc.Timeout

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		if err = api.HomeRegion(cc.VIN); err == nil {
			v.Provider = vw.NewProvider(api, cc.VIN, cc.Cache)
		}
	}

	return v, err
}
