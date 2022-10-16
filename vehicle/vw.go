package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/store"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vag/mbb"
	"github.com/evcc-io/evcc/vehicle/vag/service"
	"github.com/evcc-io/evcc/vehicle/vag/tokenrefreshservice"
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
	registry.AddWithStore("vw", NewVWFromConfig)
}

// NewVWFromConfig creates a new vehicle
func NewVWFromConfig(factory store.Provider, other map[string]interface{}) (api.Vehicle, error) {
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

	log := util.NewLogger("vw").Redact(cc.User, cc.Password, cc.VIN)

	trsStore := factory("vw.tokens.trs." + cc.User)
	trs := tokenrefreshservice.New(log, vw.TRSParams).WithStore(trsStore)

	mbbStore := factory("vw.tokens.mbb." + cc.User)
	mbb := mbb.New(log, vw.AuthClientID).WithStore(mbbStore)

	ts, err := service.MbbTokenSource(log, trs, mbb, vw.AuthParams, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	api := vw.NewAPI(log, ts, vw.Brand, vw.Country)
	api.Client.Timeout = cc.Timeout

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		if err = api.HomeRegion(cc.VIN); err == nil {
			v.Provider = vw.NewProvider(api, cc.VIN, cc.Cache)
		}
	}

	return v, err
}
