package vehicle

import (
	"fmt"
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/util/request"
	"github.com/evcc-io/evcc/vehicle/audi/etron"
	"github.com/evcc-io/evcc/vehicle/id"
	"github.com/evcc-io/evcc/vehicle/vw"
)

// https://github.com/TA2k/ioBroker.vw-connect
// https://github.com/arjenvrh/audi_connect_ha/blob/master/custom_components/audiconnect/audi_services.py

// Etron is an api.Vehicle implementation for Etron cars
type Etron struct {
	*embed
	*id.Provider // provides the api implementations
}

func init() {
	registry.Add("etron", NewEtronFromConfig)
}

// NewEtronFromConfig creates a new vehicle
func NewEtronFromConfig(other map[string]interface{}) (api.Vehicle, error) {
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

	v := &Etron{
		embed: &cc.embed,
	}

	log := util.NewLogger("etron").Redact(cc.User, cc.Password, cc.VIN)
	identity := vw.NewIdentity(log, "", etron.AuthParams, cc.User, cc.Password)

	err := identity.Login()
	if err != nil {
		return v, fmt.Errorf("login failed: %w", err)
	}

	etronIdentity := etron.NewIdentity(log, identity)
	api := etron.NewAPI(log, etronIdentity)

	cc.VIN, err = ensureVehicle(cc.VIN, api.Vehicles)

	if err == nil {
		idApi := id.NewAPI(log, identity)
		fmt.Println(idApi.Status(cc.VIN))

		v.Provider = id.NewProvider(idApi, cc.VIN, cc.Cache)
	}

	return v, err
}
