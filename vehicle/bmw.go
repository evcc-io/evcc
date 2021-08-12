package vehicle

import (
	"strings"
	"time"

	"github.com/andig/evcc/api"
	"github.com/andig/evcc/util"
	"github.com/andig/evcc/vehicle/bmw"
)

// BMW is an api.Vehicle implementation for BMW and Mini cars
type BMW struct {
	*embed
	*bmw.Provider // provides the api implementations
}

func init() {
	registry.Add("bmw", NewBMWFromConfig)
	registry.Add("mini", NewBMWFromConfig)
}

// NewBMWFromConfig creates a new vehicle
func NewBMWFromConfig(other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed               `mapstructure:",squash"`
		User, Password, VIN string
		Cache               time.Duration
	}{
		Cache: interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	v := &BMW{
		embed: &cc.embed,
	}

	log := util.NewLogger("bmw")
	identity := bmw.NewIdentity(log)

	if err := identity.Login(cc.User, cc.Password); err != nil {
		return nil, err
	}

	api := bmw.NewAPI(log, identity)

	var err error
	if cc.VIN == "" {
		cc.VIN, err = findVehicle(api.Vehicles())
		if err == nil {
			log.DEBUG.Printf("found vehicle: %v", cc.VIN)
		}
	}

	v.Provider = bmw.NewProvider(api, strings.ToUpper(cc.VIN), cc.Cache)

	return v, err
}
