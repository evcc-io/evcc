package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/logx"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/seat"
	"github.com/evcc-io/evcc/vehicle/vw"
)

// https://github.com/trocotronic/weconnect
// https://github.com/TA2k/ioBroker.vw-connect

// Seat is an api.Vehicle implementation for Seat cars
type Seat struct {
	*embed
	*vw.Provider // provides the api implementations
}

func init() {
	registry.Add("seat", NewSeatFromConfig)
}

// NewSeatFromConfig creates a new vehicle
func NewSeatFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &Seat{
		embed: &cc.embed,
	}

	log := logx.Redact(logx.NewModule("seat"), cc.User, cc.Password, cc.VIN)

	identity := vw.NewIdentity(log, seat.AuthClientID, seat.AuthParams, cc.User, cc.Password)

	err := identity.Login()
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	api := vw.NewAPI(log, identity, seat.Brand, seat.Country)
	api.Client.Timeout = cc.Timeout

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		if err = api.HomeRegion(cc.VIN); err == nil {
			v.Provider = vw.NewProvider(api, cc.VIN, cc.Cache)
		}
	}

	return v, err
}
