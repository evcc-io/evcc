package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/api/store"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/vag/loginapps"
	"github.com/evcc-io/evcc/vehicle/vag/service"
	"github.com/evcc-io/evcc/vehicle/vw/id"
)

// https://github.com/TA2k/ioBroker.vw-connect

// ID is an api.Vehicle implementation for ID cars
type ID struct {
	*embed
	*id.Provider // provides the api implementations
}

func init() {
	registry.AddWithStore("id", NewIDFromConfig)
}

// NewIDFromConfig creates a new vehicle
func NewIDFromConfig(factory store.Provider, other map[string]interface{}) (api.Vehicle, error) {
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

	v := &ID{
		embed: &cc.embed,
	}

	log := util.NewLogger("id").Redact(cc.User, cc.Password, cc.VIN)

	appsStore := factory("vw.id.tokens.loginapps." + cc.User)
	apps := loginapps.New(log).WithStore(appsStore)

	ts, err := service.LoginAppsServiceTokenSource(log, apps, id.LoginURL, id.AuthParams, cc.User, cc.Password)
	if err != nil {
		return nil, err
	}

	api := id.NewAPI(log, ts)
	api.Client.Timeout = cc.Timeout

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		v.Provider = id.NewProvider(api, cc.VIN, cc.Cache)
	}

	return v, err
}
