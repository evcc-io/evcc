package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/seat"
	"github.com/evcc-io/evcc/vehicle/vag/service"
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

	if cc.User == "" || cc.Password == "" {
		return nil, api.ErrMissingCredentials
	}

	v := &Seat{
		embed: &cc.embed,
	}

	log := util.NewLogger("seat").Redact(cc.User, cc.Password, cc.VIN)

	trs, err := service.TokenRefreshServiceTokenSource(log, seat.TRSParams, seat.AuthParams, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	ts := service.MbbTokenSource(log, trs, seat.AuthClientID)
	api := vw.NewAPI(log, ts, seat.Brand, seat.Country)
	api.Client.Timeout = cc.Timeout

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		if err = api.HomeRegion(cc.VIN); err == nil {
			v.Provider = vw.NewProvider(api, cc.VIN, cc.Cache)
		}
	}

	return v, err
}
