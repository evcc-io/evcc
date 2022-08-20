package vehicle

import (
	"time"

	"github.com/evcc-io/evcc/api"
	"github.com/evcc-io/evcc/util"
	"github.com/evcc-io/evcc/vehicle/renault"
	"github.com/evcc-io/evcc/vehicle/renault/gigya"
	"github.com/evcc-io/evcc/vehicle/renault/kamereon"
	"github.com/evcc-io/evcc/vehicle/renault/keys"
)

// Credits to
//  https://github.com/hacf-fr/renault-api
//  https://github.com/edent/Renault-Zoe-API/issues/18
//  https://github.com/epenet/Renault-Zoe-API/blob/newapimockup/Test/MyRenault.py
//  https://github.com/jamesremuscat/pyze
//  https://muscatoxblog.blogspot.com/2019/07/delving-into-renaults-new-api.html

// Renault is an api.Vehicle implementation for Renault cars
type Renault struct {
	*embed
	*renault.Provider
}

func init() {
	registry.Add("dacia", func(other map[string]interface{}) (api.Vehicle, error) {
		return NewRenaultDaciaFromConfig("dacia", other)
	})
	registry.Add("renault", func(other map[string]interface{}) (api.Vehicle, error) {
		return NewRenaultDaciaFromConfig("renault", other)
	})
}

// NewRenaultDaciaFromConfig creates a new Renault/Dacia vehicle
func NewRenaultDaciaFromConfig(brand string, other map[string]interface{}) (api.Vehicle, error) {
	cc := struct {
		embed                       `mapstructure:",squash"`
		User, Password, Region, VIN string
		Cache                       time.Duration
	}{
		Region: "de_DE",
		Cache:  interval,
	}

	if err := util.DecodeOther(other, &cc); err != nil {
		return nil, err
	}

	log := util.NewLogger(brand).Redact(cc.User, cc.Password, cc.VIN)

	v := &Renault{
		embed: &cc.embed,
	}

	keys := keys.New(log)
	keys.Load(cc.Region)

	identity := gigya.NewIdentity(log, keys.Gigya)
	if err := identity.Login(cc.User, cc.Password); err != nil {
		return nil, err
	}

	api := kamereon.New(log, keys.Kamereon, identity, func() error {
		return identity.Login(cc.User, cc.Password)
	})

	accountID, err := api.Person(identity.PersonID, brand)

	var car kamereon.Vehicle
	if err == nil {
		cc.VIN, car, err = ensureVehicleWithFeature(cc.VIN,
			func() ([]kamereon.Vehicle, error) {
				return api.Vehicles(accountID)
			},
			func(v kamereon.Vehicle) (string, kamereon.Vehicle) {
				return v.VIN, v
			},
		)
	}

	if err == nil {
		err = car.Available()
	}

	v.Provider = renault.NewProvider(api, accountID, cc.VIN, cc.Cache)

	return v, err
}
