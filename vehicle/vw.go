package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logx"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vw"
)

// https://github.com/trocotronic/weconnect
// https://github.com/TA2k/ioBroker.vw-connect

// VW is an api.Vehicle implementation for VW cars
type VW struct {
	*embed
	*vw.Provider // provides the api implementations
}

func init() {
	registry.Add("vw", NewVWFromConfig)
}

// NewVWFromConfig creates a new vehicle
func NewVWFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &VW{
		embed: &cc.embed,
	}

	log := logx.Redact(logx.NewModule("vw"), cc.User, cc.Password, cc.VIN)

	identity := vw.NewIdentity(log, vw.AuthClientID, vw.AuthParams, cc.User, cc.Password)
	err := identity.Login()
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := vw.NewAPI(log, identity, vw.Brand, vw.Country)
	api.Client.Timeout = cc.Timeout

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		if err = api.HomeRegion(cc.VIN); err == nil {
			v.Provider = vw.NewProvider(api, cc.VIN, cc.Cache)
		}
	}

	return v, err
}
