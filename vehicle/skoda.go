package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/skoda"
	"github.com/evcc-io/evcc/vehicle/vag/service"
	"github.com/evcc-io/evcc/vehicle/vag/tokenrefreshservice"
	"github.com/evcc-io/evcc/vehicle/vw"
)

// https://github.com/trocotronic/weconnect
// https://github.com/TA2k/ioBroker.vw-connect

// Skoda is an api.Vehicle implementation for Skoda cars
type Skoda struct {
	*embed
	*vw.Provider // provides the api implementations
}

func init() {
	registry.Add("skoda", NewSkodaFromConfig)
}

// NewSkodaFromConfig creates a new vehicle
func NewSkodaFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &Skoda{
		embed: &cc.embed,
	}

	log := util.NewLogger("skoda").Redact(cc.User, cc.Password, cc.VIN)

	trs := tokenrefreshservice.New(log, skoda.TRSParams)
	ts, err := service.MbbTokenSource(log, trs, skoda.AuthClientID, skoda.AuthParams, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	api := vw.NewAPI(log, ts, skoda.Brand, skoda.Country)
	api.Client.Timeout = cc.Timeout

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		if err = api.HomeRegion(cc.VIN); err == nil {
			v.Provider = vw.NewProvider(api, cc.VIN, cc.Cache)
		}
	}

	return v, err
}
